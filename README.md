Go Event Emitter
================

Simple event emitter implemented in Golang. Although there are other implementations,
this one avoids using any explicit locking (no usage of Mutex). It's purely based on
goroutines and channels.

Roadmap
=======

- [X] Initial implementation
- [ ] Test cases
- [ ] Documentation
- [ ] Benchmark
