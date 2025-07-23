import { Monitor } from "lucide-react";

interface AppIconProps {
  appName: string;
  iconPath?: string;
  className?: string;
}

export function AppIcon({
  appName,
  iconPath,
  className = "w-4 h-4",
}: AppIconProps) {
  // If we have a real extracted icon (base64 data URL), use it
  if (iconPath && iconPath.startsWith("data:image/")) {
    return (
      <img
        src={iconPath}
        alt={`${appName} icon`}
        className={`${className} object-contain`}
        onError={(e) => {
          // If the image fails to load, hide it and show fallback
          e.currentTarget.style.display = "none";
        }}
      />
    );
  }

  const getAppIcon = () => {
    const iconMap: Record<string, string> = {
      chrome: "🌐",
      firefox: "🦊",
      edge: "🌐",
      msedge: "🌐",
      vscode: "💻",
      code: "💻",
      notepad: "📝",
      explorer: "📁",
      cmd: "⚫",
      powershell: "💙",
      discord: "💬",
      spotify: "🎵",
      steam: "🎮",
      teams: "💼",
      outlook: "📧",
      word: "📄",
      excel: "📊",
      powerpoint: "📽️",
      photoshop: "🎨",
      illustrator: "✏️",
      figma: "🎨",
      slack: "💬",
      zoom: "📹",
      obs: "🎥",
      vlc: "🎬",
    };

    const lowerName = appName.toLowerCase();
    for (const [key, emoji] of Object.entries(iconMap)) {
      if (lowerName.includes(key)) {
        return emoji;
      }
    }

    return null;
  };

  const emoji = getAppIcon();

  if (emoji) {
    return (
      <span
        className={`${className} flex items-center justify-center text-sm select-none`}
      >
        {emoji}
      </span>
    );
  }

  // Final fallback to a generic icon
  return <Monitor className={`${className} text-primary flex-shrink-0`} />;
}
