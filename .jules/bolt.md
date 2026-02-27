# Bolt's Journal - Performance Learning Log

## 2024-05-23 - Regex Compilation in Hot Paths
**Learning:** Found multiple instances where `regexp.Compile` or `regexp.MustCompile` were used inside message handlers or loops. Since these handlers run for every incoming message in groups, repeated compilation is a significant CPU bottleneck and causes unnecessary allocations.
**Action:** Always pre-compile regexes as package-level variables for protection handlers and hot paths.

## 2024-05-23 - Slice Allocation in Dispatcher
**Learning:** The central message dispatcher was creating a new slice of handler functions for every single message.
**Action:** Move static handler slices to package-level variables to eliminate per-update allocations.
