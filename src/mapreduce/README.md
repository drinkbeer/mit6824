## Notes for Lab1

### Part I: Map/Reduce input and output

---
Before proceeding to Map/Reduce, we need fix some key components of the source code: 

1. `doMap()` (in `common_map.go`) - slicing the input files into intermedia files
2. `doReduce()` (in `common_reduce.go`) - read the intermedia files, sort key/value pairs by key, and reduce for each key.

##### doMap()
For each map task, it will do the following:
1. Read from input file.
2. Parse the input file into key/value pairs through self-defined mapF.
3. Generate nReduce intermedia files.
3. Write the key/value pairs to intermedia files based of hash of key.

```go
func doMap(
	jobName string, // the name of the MapReduce job
	mapTask int, // which map task this is
	inFile string,
	nReduce int, // the number of reduce task that will be run ("R" in the paper)
	mapF func(filename string, contents string) []KeyValue,
) {
	/*
		1. Read the input file line by line.
		2. Calculate the hash partition of the value using ihash(s) method.
		3. Get the reduce file name using reduceName(jobName, mapTask, r) method
		4. Write value to the reduce file
	*/

	// Read the input file, put the contents into a KeyValue array
	inContent, err := ioutil.ReadFile(inFile)
	if err != nil {
		log.Fatal("doMap failed to read input file, err: ", err)
	}

	keyValues := mapF(inFile, string(inContent))
	tasks := make([][]KeyValue, nReduce)

	for i := 0; i < len(keyValues); i++ {
		taskNo := ihash(keyValues[i].Key) % nReduce
		tasks[taskNo] = append(tasks[taskNo], keyValues[i])
	}

	// Create the reduce file, and put the KVs with the same key into one reduce file
	for i := 0; i < nReduce; i++ {
		reduceFileName := reduceName(jobName, mapTask, i)
		fmt.Printf("reduceFileName: %v\n", reduceFileName)
		file, err := os.Create(reduceFileName)
		if err != nil {
			log.Fatal("doMap failed to create reduce file, err: ", err)
		}
		defer file.Close()

		enc := json.NewEncoder(file)
		for _, kv := range tasks[i] {
			err := enc.Encode(&kv)
			if err != nil {
				log.Fatal("doMap failed to encode KeyValue pair, err: ", err)
			}
		}
	}
}
```
##### doReduce()
The process of doReduce function:
1. Read the intermedia files.
2. Decode the files into key/value pairs.
3. Count the number of keys and put the count result in a map.
4. Write the result into a final output file.

```go
func doReduce(
	jobName string, // the name of the whole MapReduce job
	reduceTask int, // which reduce task this is
	outFile string, // write the output here
	nMap int, // the number of map tasks that were run ("M" in the paper)
	reduceF func(key string, values []string) string,
) {

	/*
		1. Get the reduce file name through reduceName method
		2. Decode the reduce file to KV pairs
		3. Encode KV pairs to output file
	*/

	// Decode KV pairs from reduceFile
	kvs := make(map[string][]string)
	for m := 0; m < nMap; m++ {
		reduceFileName := reduceName(jobName, m, reduceTask)
		reduceFile, err := os.Open(reduceFileName)
		if err != nil {
			log.Fatal("doReduce failed to open, err: ", err)
		}
		defer reduceFile.Close()

		dec := json.NewDecoder(reduceFile)
		for {
			var kv KeyValue
			err := dec.Decode(&kv)
			if err != nil {
				break
			}
			kvs[kv.Key] = append(kvs[kv.Key], kv.Value)
		}
	}

	// Encode KV pairs to output file
	file, err := os.Create(outFile)
	if err != nil {
		log.Fatal("doReduce failed to create output file, err: ", err)
	}
    defer file.Close()
	enc := json.NewEncoder(file)
	for k, v := range kvs {
		enc.Encode(KeyValue{k, reduceF(k, v)})
	}

}
```

##### Testing

```
go test -run Sequential
sh test-mr.sh
```

#### Part II: Single-worker word count

---
Implement the word count example in MapReduce paper.

1. `mapF`, `reduceF` (int `wc.go`)

##### mapF()

```go
func mapF(filename string, contents string) []mapreduce.KeyValue {
	f := func(c rune) bool {
		return !unicode.IsLetter(c)
	}
	words := strings.FieldsFunc(contents, f)
	var keyValuePairs []mapreduce.KeyValue
	for _, w := range words {
		kv := mapreduce.KeyValue{w, "1"}
		keyValuePairs = append(keyValuePairs, kv)
	}
	return keyValuePairs
}
```

##### reduceF()
```go
func reduceF(key string, values []string) string {
	return strconv.Itoa(len(values))
}
```

##### Testing
```go
sh test-mr.sh
```

### Part III: Distributing MapReduce tasks

---
In part III, we will assign each task to a group of worker thread, and run in multi-core computers in parallel. We will use gRPC to run the tasks, just simulate the scenario in distributed environment.
We will implement:

1. `schedule()` (int `schedule.go`)

Each task will call `schedule()` twice. One for map phase, and one for reduce phase. `schedule()` will assign tasks to workers. Usually number of tasks is more than number of workers. 
So `schedule()` will assign each worker a task, and waiting until the completion of all tasks, and return result.

##### schedule()
`schedule()` will 
1. Get a worker from `registerChan`. Each worker includes RPC address string.
2. Execute each task.
    1. Register the task in `WaitGroup`.
    2. Create `DoTaskArgs` which contains RPC call args.
    3. `call()` executes the PRC call.
    4. Mark the task as done in `WaitGroup`.
3. Block the thread until all tasks in `WaitGroup` are done.

```go
func schedule(jobName string, mapFiles []string, nReduce int, phase jobPhase, registerChan chan string) {
	var ntasks int
	var nother int // number of inputs (for reduce) or outputs (for map)
	switch phase {
	case mapPhase:
		ntasks = len(mapFiles)
		nother = nReduce
	case reducePhase:
		ntasks = nReduce
		nother = len(mapFiles)
	}

	fmt.Printf("Schedule: %v %v tasks (%d I/Os)\n", ntasks, phase, nother)

	// All ntasks tasks have to be scheduled on workers. Once all tasks
	// have completed successfully, schedule() should return.
	//
	// Your code here (Part III, Part IV).
	//

	/*
		1. Get worker from registerChan
		2. Call "DoTask" service using RPC for each task
		3. Return the worker back to registerChan after using it
		4. Do not return until all the tasks are finished
	*/
	var wg sync.WaitGroup
	for i := 0; i < ntasks; i++ {
		doTaskArgs := DoTaskArgs{
			JobName:       jobName,
			File:          mapFiles[i],
			Phase:         phase,
			TaskNumber:    i,
			NumOtherPhase: nother}
		wg.Add(1)
		go func(doTaskArgs DoTaskArgs, registerChan chan string) {
			// get one worker from registerChan if available, block if worker not available
			worker := <-registerChan

			callResult := call(worker, "Worker.DoTask", doTaskArgs, nil)

			if callResult == false {
				fmt.Printf("Schedule: failed to call Worker %s DoTask in RPC\n", worker)
			}

			// put worker back after using it
			go func() {
				registerChan <- worker
			}()

			wg.Done()
			return
		}(doTaskArgs, registerChan)
	}
	wg.Wait()

	fmt.Printf("Schedule: %v done\n", phase)
}
```

##### Testing
```go
go test -run TestBasic
sh test-mr.sh
```

### Part IV: Handling worker failures
Let the MapReduce can handle a single-worker failure. Failures in one worker doesn't mean the whole task failed. It could be caused by reply lost, Master RPC timeout.

1. Modify the `schedule()`

```go
		go func(doTaskArgs DoTaskArgs, registerChan chan string) {
			defer wg.Done()

			for {
				// get one worker from registerChan if available, block if worker not available
				worker := <-registerChan

				callResult := call(worker, "Worker.DoTask", doTaskArgs, nil)

				// put worker back after using it
				go func() {
					registerChan <- worker
				}()

				if callResult == false {
					fmt.Printf("Schedule: failed to call Worker %s DoTask in RPC\n", worker)
				} else {
					break
				}
			}

			return
		}(doTaskArgs, registerChan)
```

##### Testing

```go
go test -run Failure
sh test-mr.sh
```

### Part V: Inverted index generation (optional)

1. `mapF()` and `reduceF()` in `ii.go`

`mapF()` slices the content of a file into <word, file_name> key/value pair, and append the pair to a `[]mapreduce.KeyValue`.

```go
func mapF(document string, value string) (res []mapreduce.KeyValue) {
	// Your code here (Part V).
	f := func(c rune) bool {
		return !unicode.IsLetter(c)
	}
	words := strings.FieldsFunc(value, f)
	// using a hashmap to remove duplicated words in one file
	wordsMap := make(map[string]string, 0)
	for _, w := range words {
		wordsMap[w] = document
	}
	res = make([]mapreduce.KeyValue, 0, len(wordsMap))
	for k, v := range wordsMap {
		kv := mapreduce.KeyValue{k, v}
		res = append(res, kv)
	}
	return res
}
```

`reduceF()` processes the values array of a specified key string. It will concatenate the files names with `,`, and output the big string.
```go
func reduceF(key string, values []string) string {
	// Your code here (Part V).
	nDocument := len(values)
	sort.Strings(values)
	var buf bytes.Buffer
	buf.WriteString(strconv.Itoa(len(values)))
	buf.WriteRune(' ')
	for i, document := range values {
		buf.WriteString(document)
		if (i != nDocument - 1) {
			buf.WriteRune(',')
		}
	}
	return buf.String()
}
```


##### Testing
```go
./test-ii.sh
sh test-mr.sh
```

#### Reference
1. [Lab 1: MapReduce](https://pdos.csail.mit.edu/6.824/labs/lab-mr.html)
2. http://www.cnblogs.com/YaoDD/p/6073794.html
3. http://blog.csdn.net/bysui/article/details/52128221
4. http://www.jianshu.com/p/bfb4aee7a827
5. http://gaocegege.com/Blog/distributed%20system/ds-lab1
