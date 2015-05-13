# Merge Trace

Merges two trace files. The second file is expected to have two columns, the first a timestamp and the second an event. The first file is expected to have a timestamp in the first column and other values in other columns. The second file's event will be appended to the first file as a new column. The timestamps are lined up and the events are duplicated.

## Example

First file:

```
Timestamp,Cores,Memory
1,0.5,100
2,0.4,110
3,0.3,120
4,0.4,130
```

Second file:

```
Timestamp,Pods
2,4
4,5
```

Merged file:

```
Timestamp,Cores,Memory,Pods
1,0.5,100,
2,0.4,110,4
3,0.3,120,4
4,0.4,130,5
```
