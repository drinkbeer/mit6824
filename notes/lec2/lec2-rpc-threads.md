# Notes for Lec2

## Infrastructure: RPC and threads

---

#### Why Go?
1. good support for threads
2. convenient RPC
3. type safe
4. garbage-collected (no use after freeing problems)
5. threads + GC is particularly attractive!
6. relatively simple
7. After the tutorial, further reading https://golang.org/doc/effective_go.html

#### Threads
* a useful structuring tool, but can be tricky
* Go calls them goroutines; everyone else calls them threads

#### Thread = "thread of execution"
* threads allow one program to do many things at once
* each thread executes serially, just like an ordinary non-threaded program
* the threads share memory
* each thread includes some per-thread state: program counter, registers, stack

#### Why threads?
* They express currency, which you need in distributed systems
* I/O concurrency
  * Client sends requests to many servers in parallel and waits for replies.
  * Server processes multiple client requests; each request may block.
  * While waiting for the disk to read data for client X, process a request from client Y.
  * Multicore performance
    * Execute code in parallel on several cores.
  * Convenience
    * In background, once per second, check whether each worker is still alive
  * Question: Is the overhead worth it? Yes. The overhead (CPU cost) is a little.

#### Is there an alternative to threads?
* Yes. Write code that explicitly interleaves activities, in a single thread. (Event-driven programming)
* Keep a table of state about each activities, e.g. each client request.
* One "event" loop that:
  * checks for new input for each activity, e.g. arrival of reply from server
  * does the next step for each activity
  * updates state
* Event-driven gets you I/O concurrency,
  * and eliminates thread costs (which can be substantial),
  * but doesn't get multi-core speedup,
  * and is painful to program (unnature)

#### Thread challenges:
* Shared data
  * e.g. what if two threads do n = n + 1 at the same time?
  * or one thread reads while another increments?
  * This is called `race` -- and is usually a bug
  * -> use locks (Go's sync.Mutex)
  * -> or avoid sharing mutable data
* Coordination between threads
  * e.g. one thread is producing data, another thread is consuming it
  * how can the consumer wait (and release the CPU)?
  * how can the producer wake up the consumer?
  * -> use Go channels or sync.Cond or WaitGroup
* Deadlock
  * Cycles via locks and/or communication (e.g. RPC or Go channels)

## CASE STUDY: Web Crawler

---
#### What is web crawler?
* Goal is to fetch all web pages, e.g. to feed to an indexer
* Web pages and links from a graph
* Multiple links to some pages
* Graph has cycles

#### Crawler challenges
* Exploit I/O concurrency
  * Network latency is more limiting than network capacity
  * Fetch many URLs at the same time
    * To increase URLs fetched per second
  * -> Need threads for concurrency
* Fetch each URL only **once**
  * avoid wasting network bandwith
  * be nice to remote servers
  * -> Need to remember which URLs visited
* Know when finished

There are two styls of solutions [crawler.go on schedule pages]

#### Serial crawler:
* performs depth-first exploration via recursive Serial calls
* The "fetched" map avoids repeats, breaks cycles
  * A single map, passed by reference, caller sees callee's updates
* but: fetches only one page at a time
  * can we just put a "go" in front of the Serial() call?
  * let's try it ... what happened? (result: doesn't work if only add go)

#### ConcurrentMutex crawler:
* Creates a thread for each page fetched
  * Many concurrent fetches, higher fetch rate
* The "go" 