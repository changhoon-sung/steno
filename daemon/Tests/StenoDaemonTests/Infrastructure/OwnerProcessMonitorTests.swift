import Foundation
import Synchronization
import Testing

@testable import StenoDaemon

@Suite("TUI owner lifetime")
struct OwnerProcessMonitorTests {
    @Test(arguments: [SIGTERM, SIGKILL])
    func ownerExitIsDetected(signal: Int32) async throws {
        let owner = Process()
        owner.executableURL = URL(fileURLWithPath: "/bin/sleep")
        owner.arguments = ["60"]
        try owner.run()
        defer {
            if owner.isRunning { kill(owner.processIdentifier, SIGKILL) }
            owner.waitUntilExit()
        }

        let exited = Mutex(false)
        let monitor = try OwnerProcessMonitor(pid: owner.processIdentifier) {
            exited.withLock { $0 = true }
        }
        defer { withExtendedLifetime(monitor) {} }

        // A live owner must not trigger shutdown.
        try await Task.sleep(for: .milliseconds(50))
        #expect(!exited.withLock { $0 })
        #expect(kill(owner.processIdentifier, signal) == 0)

        let deadline = ContinuousClock.now + .seconds(2)
        while !exited.withLock({ $0 }), ContinuousClock.now < deadline {
            try await Task.sleep(for: .milliseconds(10))
        }
        #expect(exited.withLock { $0 })
    }

    @Test(arguments: [Int32(-1), 0, 1, Int32.max])
    func invalidOrMissingOwnerIsRejected(pid: Int32) {
        #expect(throws: (any Error).self) {
            _ = try OwnerProcessMonitor(pid: pid) {}
        }
    }

    @Test func runCommandAcceptsOwnerPID() throws {
        let command = try RunCommand.parse(["--owner-pid", "1234"])
        #expect(command.ownerPID == 1234)
        #expect(try RunCommand.parse([]).ownerPID == nil)
    }

    @Test func alreadyExitedOwnerIsRejected() throws {
        let owner = Process()
        owner.executableURL = URL(fileURLWithPath: "/usr/bin/true")
        try owner.run()
        owner.waitUntilExit()
        #expect(throws: (any Error).self) {
            _ = try OwnerProcessMonitor(pid: owner.processIdentifier) {}
        }
    }
}
