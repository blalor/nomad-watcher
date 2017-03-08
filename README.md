Who watches the watcher?

This is a simple service that watches [Nomad](https://nomadproject.io)'s nodes, jobs, allocations, and evaluations, and writes the events to a file.  The intention is that they'll be shipped to a central log collection system so users and operators can get insight into what Nomad is doing.

## usage

    nomad-watcher --events-file=nomad_events.json

## sample events

prettified; the actual log file contains one event per line.

### eval

```json
{
  "@timestamp": "2017-03-08T05:50:02.94624366Z",
  "wait_index": 344274,
  "eval": {
    "ID": "4efd75c0-d183-e535-53cb-f223fe97f821",
    "Priority": 50,
    "Type": "service",
    "TriggeredBy": "job-deregister",
    "JobID": "some-job/periodic-1488936600",
    "JobModifyIndex": 344269,
    "NodeID": "",
    "NodeModifyIndex": 0,
    "Status": "complete",
    "StatusDescription": "",
    "Wait": 0,
    "NextEval": "",
    "PreviousEval": "",
    "BlockedEval": "",
    "FailedTGAllocs": null,
    "QueuedAllocations": {},
    "CreateIndex": 344270,
    "ModifyIndex": 344274
  }
}
```

### job

```json
{
  "@timestamp": "2017-03-08T05:45:05.341084793Z",
  "wait_index": 344261,
  "job": {
    "ID": "some-job/periodic-1488951900",
    "ParentID": "some-job",
    "Name": "some-job/periodic-1488951900",
    "Type": "batch",
    "Priority": 50,
    "Status": "dead",
    "StatusDescription": "",
    "JobSummary": {
      "JobID": "some-job/periodic-1488951900",
      "Summary": {
        "importer": {
          "Queued": 0,
          "Complete": 1,
          "Failed": 0,
          "Running": 0,
          "Starting": 0,
          "Lost": 0
        }
      },
      "Children": {
        "Pending": 0,
        "Running": 0,
        "Dead": 0
      },
      "CreateIndex": 344253,
      "ModifyIndex": 344261
    },
    "CreateIndex": 344253,
    "ModifyIndex": 344261,
    "JobModifyIndex": 344253
  }
}
```
