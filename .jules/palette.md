# Palette's Journal - UX & Design Learnings

## 2026-02-28 - Scannable System Dashboards
**Learning:** Dense technical data like `/stats` becomes overwhelming in plain text. Users scan for key metrics (Users, Uptime) before technical ones (RAM, CPU).
**Action:** Use visual separators (━━━━━━━━━━━━━━━━━━━━), bold headers, and group related data into blockquotes. Use `<blockquote expandable>` for technical details to prioritize primary metrics while keeping the message clean.

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
