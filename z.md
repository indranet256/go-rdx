`` `
This file tests the value-order of RDX elements.
That is the order Eulerian containers use; it is also used as a fallback in other orders.
```

```
An empty tuple is the leastest element.
Other tuples get ordered according to their key element (i.e. the first one).
A tuple of one element is the same thing as the element itself.
```
    (),
```
Floats get ordered numerically.
```
    -0.123,
    1.23,
    1.24 @a1ec-1,
```
Integers are like floats.
```
    -3,
    -2,
    1,
    2,
    3 @b0b-1,
```
Again, a tuple is ordered the same as its key.
```
    0:5,
    2:3,
    4:1,
```
References get ordered in the "Lamport order" (seq, then src).
```
    b0b-2,
    b0b-3 @b0b-4,
    b0b-4 @b0b-3,
    b0b-5 @a1ec-1,
    a1ec-7,
```
Strings get ordered lexicographically (strcmp wise).
```
    "Alice",
    "Bob" @b0b-1,
    "Bobby",
    "Carol",
```
Terms: same as strings.
```
    false,
    once:twice,
    one:two:three,
    true,
```
Eulerian collections: by their id.
```
    { 1, 2, 3,  },
    { @e-1 1, 2, 3 },
    { @e-2 },
```
...that applies to arrays as well...
```
    [ 1, 2, 3 ],
    [ @a-1 2, 3, 4, 5 ],
    [ @a-2 3 1,2,3 ],
```
...as well as to multiplexed collections.
```
    < 1@a1ec-1, 2@b0b-1 >,
    < @b0b-0 1@a1ec-1, 2@b0b-1 >,
    <@b0b-1 ~@b0b-2, ~@a1ec-3 >,
    <@a1ec-1 ~@b0b-3, ~@a1ec-3 >,
