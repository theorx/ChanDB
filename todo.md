## Todo list


* Research WriteAt

### API

* Insert()
* Read() 
* Length()
* GarbageCollect() <- Separate process / routine, more details later


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