# External version support (aka Zookie)
```bash
DATASTORE_PROJECT_ID=my-project go run ./example/externalzookie
```


Inserts with POST output the version, create time and update time:
```bash
$ curl -X POST "localhost:8080/task" --data '{ "name": "Foobar" }'
{"key":"Eg8KBFRhc2sQgICkt9TzqQg","name":"Foobar","version":1711378081740127,"create_time":"2024-03-25T14:48:01.740127Z","update_time":"2024-03-25T14:48:01.740127Z"}
```

Updates with PUT output the new version:
```bash
$ curl -X PUT "localhost:8080/task?key=Eg8KBFRhc2sQgICkt9TzqQg" --data '{ "name": "Foobar3", "version": 1711378081740127 }'
{"version":1711378138154815}
```

Then GET also reflects the same version, create time and update time:
```bash
$ curl -X GET "localhost:8080/task?key=Eg8KBFRhc2sQgICkt9TzqQg"
{"key":"Eg8KBFRhc2sQgICkt9TzqQg","name":"Foobar3","version":1711378138154815,"create_time":"2024-03-25T14:48:01.740127Z","update_time":"2024-03-25T14:48:58.154815Z"}
```

Updating with the wrong versions:
```bash
# too low
$ curl -X PUT "localhost:8080/task?key=Eg8KBFRhc2sQgICkt9TzqQg" --data '{ "name": "Reject me", "version": 1711378138154814 }'
conflict

# too high
$ curl -X PUT "localhost:8080/task?key=Eg8KBFRhc2sQgICkt9TzqQg" --data '{ "name": "Reject me", "version": 1711378138154814 }'
rpc error: code = InvalidArgument desc = Invalid base version, it is greater than the stored version: app: "e~my-project"
path <
  Element {
    type: "Task"
    id: 0x10a79d46e90000
  }
>
```

Deleting it also returns a version:
```bash
$ curl -X DELETE "localhost:8080/task?key=Eg8KBFRhc2sQgICkt9TzqQg" --data '{ "version": 1711378138154815 }'
{"version":1711378500803887}
```

And any further modification fails:
```bash
% curl -X PUT "localhost:8080/task?key=Eg8KBFRhc2sQgICkt9TzqQg" --data '{ "name": "Reject me", "version": 1711378500803887 }'
rpc error: code = NotFound desc = no entity to update: app: "e~my-project"
path <
  Element {
    type: "Task"
    id: 0x10a79d46e90000
  }
>
```