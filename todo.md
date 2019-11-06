## Todo list


* Research WriteAt

### API

* Insert()
* Read() 
* Length()
* GarbageCollect() <- Separate process / routine, more details later

### Implement recovery strategy, garbage collection
* Implement fsync to force changes to the disk via separate routine, 1s interval

* Program - call fsync 1 
--- user space 
* Page cache <-kernel 2

* disk-controller / raid controller cache 3

O_DIRECT

### File format

* [Marker][Json Payload][Row delimiter]


__Example of deleted rows__
```

-{"key":"value"}
-{"key":"value"}
-{"key":"value"}
-{"key":"value"}

```

__Example of mixed rows__

```

-{"key":"value"}
 {"key":"value"}
-{"key":"value"}
 {"key":"value"}
-{"key":"value"}
 {"key":"value"}
-{"key":"value"}

```