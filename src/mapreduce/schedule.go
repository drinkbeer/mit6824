package mapreduce

import (
	"fmt"
	"sync"
)

//
// schedule() starts and waits for all tasks in the given phase (mapPhase
// or reducePhase). the mapFiles argument holds the names of the files that
// are the inputs to the map phase, one per map task. nReduce is the
// number of reduce tasks. the registerChan argument yields a stream
// of registered workers; each item is the worker's RPC address,
// suitable for passing to call(). registerChan will yield all
// existing registered workers (if any) and new ones as they register.
//
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
