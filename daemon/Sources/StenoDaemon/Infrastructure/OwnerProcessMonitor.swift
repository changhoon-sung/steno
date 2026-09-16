import Foundation

/// Watches the owning TUI in the kernel, including exits caused by SIGKILL.
/// Keep this object alive for as long as the daemon is running.
final class OwnerProcessMonitor {
    enum MonitorError: Error, CustomStringConvertible {
        case ownerNotRunning(Int32)

        var description: String {
            switch self {
            case .ownerNotRunning(let pid):
                return "TUI owner PID \(pid) is invalid or no longer running"
            }
        }
    }

    private let source: any DispatchSourceProcess

    init(pid: Int32, onExit: @escaping @Sendable () -> Void) throws {
        guard pid > 1, pid != getpid(), kill(pid, 0) == 0 else {
            throw MonitorError.ownerNotRunning(pid)
        }
        source = DispatchSource.makeProcessSource(
            identifier: pid, eventMask: .exit, queue: .global()
        )
        source.setEventHandler(handler: onExit)
        source.activate()

        // Fail closed if the owner disappeared while the source was created.
        guard kill(pid, 0) == 0 else {
            source.cancel()
            throw MonitorError.ownerNotRunning(pid)
        }
    }

    deinit {
        source.cancel()
    }
}
