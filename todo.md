
### API

* Write()
* Read() 
* Length() <- is length a real thing?!?!?!?
* Close()

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