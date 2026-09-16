@preconcurrency import AVFoundation
import CoreMedia
import Foundation
import Speech

/// Real speech recognizer factory using macOS 26 SpeechAnalyzer API.
public final class DefaultSpeechRecognizerFactory: SpeechRecognizerFactory, Sendable {
    private let fastResults: Bool

    public init(fastResults: Bool = false) { self.fastResults = fastResults }

    public func makeRecognizer(locale: Locale, format: AVAudioFormat, source: AudioSourceType)
        async throws -> SpeechRecognizerHandle {
        DefaultSpeechRecognizerHandle(locale: locale, inputFormat: format, source: source, fastResults: fastResults)
    }
}

/// Real speech recognizer handle wrapping SpeechAnalyzer.
///
/// Uses `@unchecked Sendable` because `SpeechAnalyzer` and `SpeechTranscriber`
/// are not yet marked Sendable by Apple. Access is serialized: `transcribe` sets
/// up state synchronously, then a single Task drives the pipeline.
final class DefaultSpeechRecognizerHandle: SpeechRecognizerHandle, @unchecked Sendable {
    private let locale: Locale
    private let inputFormat: AVAudioFormat
    private let source: AudioSourceType
    private let fastResults: Bool
    private var analyzer: SpeechAnalyzer?
    private var transcriber: SpeechTranscriber?

    init(locale: Locale, inputFormat: AVAudioFormat, source: AudioSourceType, fastResults: Bool = false) {
        self.locale = locale
        self.inputFormat = inputFormat
        self.source = source
        self.fastResults = fastResults
    }

    func transcribe(buffers: AsyncStream<AVAudioPCMBuffer>)
        -> AsyncThrowingStream<RecognizerResult, Error> {
        let transcriber = SpeechTranscriber(
            locale: self.locale,
            transcriptionOptions: [],
            reportingOptions: fastResults ? [.volatileResults, .fastResults] : [.volatileResults],
            attributeOptions: []
        )
        self.transcriber = transcriber

        let analyzer = SpeechAnalyzer(modules: [transcriber])
        self.analyzer = analyzer

        let (inputSequence, inputBuilder) = AsyncStream<AnalyzerInput>.makeStream()

        let pipeline = Pipeline(
            buffers: buffers,
            inputFormat: inputFormat,
            source: source,
            analyzer: analyzer,
            transcriber: transcriber,
            inputSequence: inputSequence,
            inputBuilder: inputBuilder
        )

        return AsyncThrowingStream { continuation in
            let task = Task.detached {
                await pipeline.run(continuation: continuation)
            }
            continuation.onTermination = { _ in
                task.cancel()
            }
        }
    }

    func stop() async {
        do {
            try await analyzer?.finalizeAndFinishThroughEndOfInput()
        } catch {
            // Ignore cleanup errors
        }
        analyzer = nil
        transcriber = nil
    }
}

/// Packages all pipeline state into a Sendable value that can cross
/// the `sending` boundary of `Task.detached` without capture issues.
private struct Pipeline: @unchecked Sendable {
    let buffers: AsyncStream<AVAudioPCMBuffer>
    let inputFormat: AVAudioFormat
    let source: AudioSourceType
    let analyzer: SpeechAnalyzer
    let transcriber: SpeechTranscriber
    let inputSequence: AsyncStream<AnalyzerInput>
    let inputBuilder: AsyncStream<AnalyzerInput>.Continuation

    func run(continuation: AsyncThrowingStream<RecognizerResult, Error>.Continuation) async {
        let analyzerFormat = await SpeechAnalyzer.bestAvailableAudioFormat(
            compatibleWith: [transcriber]
        ) ?? inputFormat

        do {
            try await withThrowingTaskGroup(of: Void.self) { group in
                // Task 1: Feed all converted audio and flush its pending tail.
                group.addTask {
                    try await AnalyzerAudioFeeder.feed(
                        self.buffers, inputFormat: self.inputFormat,
                        analyzerFormat: analyzerFormat, into: self.inputBuilder
                    )
                }

                // Task 2: Listen for transcription results.
                // MUST start BEFORE analyzer.start() because start() blocks
                // until the input stream ends.
                group.addTask {
                    for try await result in self.transcriber.results {
                        let text = String(result.text.characters)
                        // #64: carry the result's audio time range (analyzer
                        // input timeline) so segments can be joined to
                        // diarization windows on the frame-accurate capture
                        // clock. CMTimeGetSeconds yields NaN for an invalid
                        // range → treated as unavailable (nil).
                        let startSeconds = CMTimeGetSeconds(result.range.start)
                        let durationSeconds = CMTimeGetSeconds(result.range.duration)
                        continuation.yield(RecognizerResult(
                            text: text,
                            isFinal: result.isFinal,
                            source: self.source,
                            audioStartSeconds: startSeconds.isFinite ? startSeconds : nil,
                            audioDurationSeconds: durationSeconds.isFinite ? durationSeconds : nil
                        ))
                    }
                }

                // Task 3: Start the analyzer on @MainActor.
                // SpeechAnalyzer MUST run on @MainActor — crashes with
                // SIGTRAP otherwise.
                group.addTask {
                    try await Task { @MainActor in
                        try await self.analyzer.start(inputSequence: self.inputSequence)
                    }.value
                }

                try await group.waitForAll()
            }
            continuation.finish()
        } catch {
            continuation.finish(throwing: error)
        }
    }
}
