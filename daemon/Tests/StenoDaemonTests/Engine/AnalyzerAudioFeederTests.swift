import AVFoundation
import CoreMedia
import Testing
import Speech
@testable import StenoDaemon

struct AnalyzerAudioFeederTests {
    @Test(arguments: [AVAudioFrameCount(480), 1024, 4096])
    func conversionPreservesCompleteAudioDuration(chunkFrames: AVAudioFrameCount) async throws {
        guard #available(macOS 27, *) else { return }
        let source = try #require(AVAudioFormat(commonFormat: .pcmFormatFloat32, sampleRate: 48000, channels: 2, interleaved: false))
        let target = try #require(AVAudioFormat(commonFormat: .pcmFormatInt16, sampleRate: 16000, channels: 1, interleaved: true))
        let (buffers, input) = AsyncStream<AVAudioPCMBuffer>.makeStream()
        for _ in 0..<300 {
            let buffer = try #require(AVAudioPCMBuffer(pcmFormat: source, frameCapacity: chunkFrames))
            buffer.frameLength = chunkFrames
            for channel in 0..<2 {
                buffer.floatChannelData?[channel].initialize(repeating: 0.1, count: Int(chunkFrames))
            }
            // The fixture never mutates a buffer after this transfer.
            nonisolated(unsafe) let owned = buffer
            input.yield(owned)
        }
        input.finish()
        let (converted, output) = AsyncStream<AnalyzerInput>.makeStream()
        try await AnalyzerAudioFeeder.feed(buffers, inputFormat: source, analyzerFormat: target, into: output)
        var duration = 0.0
        for await chunk in converted { duration += CMTimeGetSeconds(chunk.bufferDuration) }
        let expected = Double(chunkFrames) * 300 / 48000
        #expect(abs(duration - expected) < 1 / 16000.0)
    }
}
