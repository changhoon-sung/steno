import Foundation
import Testing
@testable import StenoDaemon

struct LanguageSettingsTests {
    @Test func savedLanguageRoundTripsAndControlsStartupLocale() throws {
        let settings = StenoSettings(lastLocale: "ko-KR")
        let data = try JSONEncoder().encode(settings)
        let loaded = try JSONDecoder().decode(StenoSettings.self, from: data)
        #expect(loaded.lastLocale == "ko-KR")
        #expect(loaded.transcriptionLocale.identifier(.bcp47) == "ko-KR")
    }

    @Test func oldSettingsKeepSystemLanguage() throws {
        let settings = try JSONDecoder().decode(StenoSettings.self, from: Data("{}".utf8))
        #expect(settings.lastLocale == nil)
        #expect(settings.transcriptionLocale == .current)
    }
}
