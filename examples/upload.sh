#!/bin/bash
curl localhost:8080/bucket/key -X PUT -H 'Content-Type: application/pdf' -H 'Expect: 100-continue' --data-binary '@./5mb.pdf' -v