#   Discount CRDT

Discount is a greatly simplified CRDT algorithm that
is as simple and resilient as possible and, importantly,
fits into the constraints of an LSM merge operator.

RDX has 128-bit element IDs consisting of a 64-bit sequence
number `seq` and a 64-bit replica id `src`. Those are essentially 
Lamport timestamps. The `L` element additionally subdivides the `seq`
into 6 bit of "revision" and 64-6=58 bit "locator". The locator
(mainly) defines the position of a member element within the 
array.

Numbering of the elements may follow several "tactics".

##  The Figma tactic

This numbering algorithm follows the Figma approach. Figma
collaborative editor is labeling elements with floats, then
sorts them numerically.
Thus, any insertion assigns a label inbetween two neighboring 
labels, e.g. `(seq_left + seq_right)/2`.
`/2` might be too risky as it allows for clashes in case we
skip `src` for efficiency. The particular ratio can be 
randomized, as one option.
RDX employs 64 bit integers, not floats, but that changes
nothing in essence.
In this tactic, a hello world may look like:
```
 [ "Hello"@2000 "world"@3000 " !"@4000 ]
```
Remember that RDX IDs get serialized in RON Base64 for efficiency.
A trivial edit of an array may look like:
```
 [ "Hello"@2000 " "@2q00 "world"@3000 " !"@4000 ]
```
In case we want to drop the annoying exclamation mark, we add
a patch:
```
 [ "Hello"@2000 " "@2q00 "world"@3000 " !"@4000 ] +
 [ ""@4001 ] = 
 [ "Hello"@2000 " "@2q00 "world"@3000 ""@4001 ]
```
Remember, odd revisions are tombstones.


##  The lazy tactic

The lazy tactic is useful when we do not plan to edit an
array much. That way, we save greatly on the metadata.
Elements have no ID at all (all IDs are zeroes).
```
[ 1 2 3 4 5 ]
```
There is no way to specify a "patch" to such an array;
the change set is the complete new state of an array
and we can only do appends or replaces.
```
[ 1 2 3 4 5 ] +
[ 1 2.0@2 3 4 5 6 ] +
[ 1 2 3 4 5 6 7 ] =
[ 1 2.0@2 3 4 5 6 7 ]
```
The good part, it is possible to use figma-patches in 
a lazily-numbered array. Within figma-sections, we can
skip data when making a patch.
```
[ 1 2 3 4 5 ] +
[ 1 2 3 3.1@100 3.2@200 4 5 ] +
[ 1 2 3 3.3@300 4 5 ] =
[ 1 2 3 3.1@100 3.2@200 3.3@300 4 5 ]
```
