# Bolt's Journal - Performance Learning Log

## 2026-02-27 - Regex Compilation in Hot Paths
**Learning:** Found multiple instances where `regexp.Compile` or `regexp.MustCompile` were used inside message handlers or loops. Since these handlers run for every incoming message in groups, repeated compilation is a significant CPU bottleneck and causes unnecessary allocations.
**Action:** Always pre-compile regexes as package-level variables for protection handlers and hot paths.

## 2026-02-27 - Slice Allocation in Dispatcher
**Learning:** The central message dispatcher was creating a new slice of handler functions for every single message.
**Action:** Move static handler slices to package-level variables to eliminate per-update allocations.

## 2026-02-27 - Lock Contention in Shared Cache
**Learning:** Using a single `sync.Mutex` for the global cache caused significant bottlenecking when multiple protection handlers (10+) checked settings concurrently for every message.
**Action:** Use `sync.RWMutex` with double-checked locking in `Get()` to allow parallel reads on the fast path.

## 2026-02-27 - Allocation-Free Cache Keys
**Learning:** Using `fmt.Sprintf("%d:%d", chatID, userID)` as a cache key in the message hot path (e.g., for `IsAdmin` checks) caused millions of unnecessary string allocations.
**Action:** Use a struct (e.g., `type adminCacheKey struct { chatID, userID int64 }`) as the map key to leverage Go's efficient struct hashing without allocations.

## 2026-02-27 - Lock-Free Global State with atomic.Pointer
**Learning:** The global bot state (served users/chats) was being fetched via a string-keyed generic cache with a 60-minute TTL. This caused repeated type assertions and potential thundering herd issues if the cache expired under high load.
**Action:** Use `atomic.Pointer` for lock-free reads and a dedicated `sync.Mutex` for lazy initialization. This ensures O(1) access without allocations or lock contention in the hot path.

## 2026-03-11 - Single-Pass Scanner for Content Filtering
**Learning:** Content filtering handlers (like NoPromo) often perform multiple passes over the message text for different checks (emojis, caps, line counts, bullet points), each potentially allocating or causing redundant traversals.
**Action:** Consolidate multiple string analysis tasks into a single-pass UTF-8 scanner to minimize CPU cycles and memory pressure in the message processing hot path.

## 2026-03-11 - Zero-Allocation Word Scanning
**Learning:** Using `strings.Fields` and `strings.ToLower` to parse words in message handlers causes O(N) heap allocations per message.
**Action:** Use a manual loop to scan words in-place and perform case-insensitive checks without allocating new strings.

## 2026-03-11 - Allocation-Free Cleanup Scheduler Keys
**Learning:** The `CleanupScheduler` used a formatted string `"chatid:msgid"` as a map key. In groups with auto-delete enabled, this resulted in thousands of heap allocations per hour just for key generation.
**Action:** Use a struct `type deletionKey struct { chatID int64; messageID int32 }` as the map key to eliminate all allocations during message scheduling and cancellation.

## 2026-03-13 - Zero-Allocation Trie Matcher with Word Boundaries
**Learning:** Standardizing word filters with `strings.Fields` and `strings.ToLower` causes heap allocations on every call. Switching to a Trie improves performance but can introduce the "Scunthorpe problem" (false positives in substrings) if not careful with boundaries.
**Action:** Implement a Trie-based matcher that processes runes on-the-fly and explicitly validates word boundaries (checking `unicode.IsLetter` before and after the match). This achieves 0 allocs/op and fixes detection of multi-word phrases.
