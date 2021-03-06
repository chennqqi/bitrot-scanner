package main

import "sync"
import "time"

var workerStartJobs chan job
var workerStartJobswg sync.WaitGroup

func initWorkerStart() {
    workerStartJobs = make(chan job, workerCount*2)
    workerStartJobswg.Add(workerCount)
    for i := 0; i < workerCount; i++ {
        go workerStart()
    }
    go func() {
        workerStartJobswg.Wait()
        close(workerIOJobs)
    }()
}

func workerStart() {
    defer workerStartJobswg.Done()
    for currentJob := range workerStartJobs {
        err := func() error {
            time_start := time.Now()
            Trace.Printf("%v: Start Processing...\n", currentJob.path)

            // Only process regular files
            if !currentJob.info.Mode().IsRegular() {
                return nil
            }

            // Skip files if they've been modified too recently
            if currentJob.mtime > (time.Now().Unix() - mtimeSettle) {
                Info.Printf("%v: Has been modified too recently. Skipping due to --mtimeSettle\n", currentJob.path)
                return nil
            }

            if skipValidation && !currentJob.missingChecksums() {
                Info.Printf("%v: Already has all checksums. Skipping due to --skipValidation\n", currentJob.path)
                return nil
            }

            if skipCreate && currentJob.checksumCount == 0 {
                Info.Printf("%v: Missing all checksums. Skipping due to --skipCreate\n", currentJob.path)
                return nil
            }

            currentJob.checksumMTime = GetMTimeXattr(currentJob.path)

            // Validate that the file hasn't received a new mtime
            if currentJob.checksumMTime != 0 && currentJob.mtime > currentJob.checksumMTime {
                Warn.Printf("%v has a mtime after checksums were generated!\n", currentJob.path)
            }

            currentJob.initalizeChecksums()
            duration := time.Since(time_start)
            currentJob.duration += duration
            Trace.Printf("%v: Start Processing took %v\n", currentJob.path, duration)
            workerIOJobs <- currentJob
            return nil
        }()

        if err != nil {
            Error.Printf("%v: Start Processing: %v\n", currentJob.path, err)
        }
    }
}
