import AVFoundation
import Speech

/// Owns conversion on the single audio-feed task. Every output, including
/// the converter's held-over tail, is delivered before the input ends.
enum AnalyzerAudioFeeder {
    enum ConversionError: Error { case unavailable, allocationFailed }

    static func feed(
        _ buffers: AsyncStream<AVAudioPCMBuffer>,
        inputFormat: AVAudioFormat,
        analyzerFormat: AVAudioFormat,
        into output: AsyncStream<AnalyzerInput>.Continuation
    ) async throws {
        defer { output.finish() }
        if #available(macOS 27, *) {
            let converter = AnalyzerInputConverter(analyzerFormat: analyzerFormat)
            for await buffer in buffers {
                try Task.checkCancellation()
                // Capture sources yield independently owned buffers. The
                // converter may retain them across successive convert calls.
                for input in try converter.convert(buffer, at: nil) {
                    output.yield(input)
                }
            }
            try Task.checkCancellation()
            for input in try converter.flush() { output.yield(input) }
            return
        }

        // Runtime compatibility for macOS 26. New builds use the macOS 27 SDK.
        guard inputFormat != analyzerFormat else {
            for await buffer in buffers {
                try Task.checkCancellation()
                output.yield(AnalyzerInput(buffer: buffer))
            }
            return
        }
        guard let converter = AVAudioConverter(from: inputFormat, to: analyzerFormat) else {
            throw ConversionError.unavailable
        }
        converter.primeMethod = .none
        for await buffer in buffers {
            try Task.checkCancellation()
            var supplied = false
            let capacity = AVAudioFrameCount(ceil(Double(buffer.frameLength) * analyzerFormat.sampleRate / inputFormat.sampleRate)) + 32
            while true {
                guard let converted = AVAudioPCMBuffer(pcmFormat: analyzerFormat, frameCapacity: capacity) else {
                    throw ConversionError.allocationFailed
                }
                var error: NSError?
                let status = converter.convert(to: converted, error: &error) { _, inputStatus in
                    guard !supplied else { inputStatus.pointee = .noDataNow; return nil }
                    supplied = true
                    inputStatus.pointee = .haveData
                    return buffer
                }
                if let error { throw error }
                if converted.frameLength > 0 { output.yield(AnalyzerInput(buffer: converted)) }
                if status != .haveData || converted.frameLength == 0 { break }
            }
        }
        // Drain the legacy converter too; cancellation is never a request to
        // flush additional audio after the owning TUI has stopped capture.
        try Task.checkCancellation()
        while true {
            guard let tail = AVAudioPCMBuffer(pcmFormat: analyzerFormat, frameCapacity: 4096) else {
                throw ConversionError.allocationFailed
            }
            var error: NSError?
            let status = converter.convert(to: tail, error: &error) { _, state in
                state.pointee = .endOfStream
                return nil
            }
            if let error { throw error }
            if tail.frameLength > 0 { output.yield(AnalyzerInput(buffer: tail)) }
            if status != .haveData || tail.frameLength == 0 { break }
        }
    }
}
