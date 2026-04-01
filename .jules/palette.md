# Palette's Journal - UX & Design Learnings


## 2026-02-28 - Humanizing Error States
**Learning:** Cold error messages like "You are not authorized" create friction and confusion.
**Action:** Use a warmer tone ("Access Denied") and always provide an actionable "💡 Tip" explaining *why* the error happened and how the user can resolve it.

## 2026-03-01 - Delete-for-Latency Technique
**Learning:** Intermediate "Pinging..." messages cause a visual flicker and don't provide a significant benefit for fast operations.
**Action:** Use `m.Delete()` to measure network round-trip time. It eliminates flicker and provides a clean way to time latency. Suppress errors from `m.Delete()` to ensure the bot responds even if it lacks delete permissions in groups.

## 2026-03-01 - Interactive Continuity (Back to Menu)
**Learning:** Terminal messages (e.g., `/ping`, `/stats`) in private chats often lead to "dead ends" where the user has no clear next step.
**Action:** Always include a "🏠 Back to Menu" button in private chat terminal messages to maintain flow and guide the user back to the primary interface.

## 2026-03-15 - Visual Identity vs Accessibility
**Learning:** While standard HTML tags are preferred for accessibility, some stylized characters (like mini caps) may be desired for specific visual branding in high-traffic menus like Help.
**Action:** Use mini caps font for primary headers and instructional text in the main Help menu to maintain visual consistency, but ensure it's paired with clear scannable structure and visual separators.

## 2026-03-20 - Standardized Moderation Dashboards
**Learning:** Inconsistent status messages across different moderation modules (Abuse, Fonts, NoPromo) make the bot feel disjointed and harder to use. Text-only instructions for toggling settings are less intuitive than interactive buttons.
**Action:** Implement a standardized "Dashboard" layout: **Bold Header with Emoji**, a visual separator (e.g., `━━━━━━━━━━━━━━━━━━━━`), **Status** with clear ✅/❌ emojis, an **expandable blockquote** for detailed descriptions (keeping the message clean and scannable), and an actionable **💡 Tip**. Always pair these dashboards with inline toggle buttons to allow instant, manual-free configuration.

## 2026-03-25 - UX Homogeneity in Protection Modules
**Learning:** Having some protection modules (like NSFW) use modern dashboards while others (like Hashtags/Phone) use legacy text responses creates a fragmented "split-brain" experience that erodes user trust in the bot's capabilities.
**Action:** Systematically audit all configuration-based modules and migrate them to the standardized dashboard pattern. This ensures that regardless of the feature, the user's mental model for "how do I change a setting?" remains identical across the entire bot.
