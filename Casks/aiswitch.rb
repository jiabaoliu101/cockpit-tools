cask "aiswitch" do
  version "1.0.0"
  sha256 "9d5d35d6a5dba1c809cd88a99034cf9e55abe8a7f8f1a64d84c938289ab72698"

  url "https://github.com/jiabaoliu101/aiswitch/releases/download/v#{version}/AISwitch_#{version}_universal.dmg",
      verified: "github.com/jiabaoliu101/aiswitch/"
  name "AISwitch"
  desc "AI IDE account switching tool with subscription management"
  homepage "https://github.com/jiabaoliu101/aiswitch"

  auto_updates true

  postflight do
    system_command "/usr/bin/xattr",
                   args: ["-cr", "#{appdir}/AISwitch.app"],
                   sudo: true
  end

  app "AISwitch.app"

  zap trash: [
    "~/Library/Application Support/com.jiabaoliu.aiswitch",
    "~/Library/Caches/com.jiabaoliu.aiswitch",
    "~/Library/Preferences/com.jiabaoliu.aiswitch.plist",
    "~/Library/Saved Application State/com.jiabaoliu.aiswitch.savedState",
  ]

  caveats <<~EOS
    The app is automatically quarantined by macOS. A postflight hook has been added to remove this quarantine.
    If you still encounter the "App is damaged" error, please run:
      sudo xattr -rd com.apple.quarantine "/Applications/AISwitch.app"
  EOS
end
