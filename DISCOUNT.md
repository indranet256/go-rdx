#   Discontinuity CRDT (DISCONT)

DISCONT is a greatly simplified CRDT algorithm that
is as simple and resilient as possible and, importantly,
fits into the constraints of an LSM merge operator.
DISCONT replaces CausalTree/RGA for Linear RDX elements.

The DISCONT algorithm is essentially merge sort using the
order of element IDs. Implementation-wise, it is a merge
sort using a heap of iterators. This technique is the norm
in the LSM world. On top of that, it has a little twist.
Which is more like a roundhouse kick, see below.

RDX has 128-bit element IDs consisting of a 64-bit sequence
number `seq` and a 64-bit replica id `src`. Those are essentially 
Lamport timestamps. The array type (`L`) additionally subdivides the `seq`
into 6 bit of "revision" and 64-6=58 bit "locator". The locator
(mainly) defines the position of a member element within the 
array.

When merging versions of the state and patches, DISCONT
creates an iterator for each input, puts iterators in a heap
and... iterates like a merge sort should do. Importantly,
equal inputs get merged (two versions of an array would
normally have lots of identical elements).

Most of the magic hides in the algorithms of element numbering.
Here we may follow various "tactics".

##  The Figma tactic

This numbering algorithm follows the Figma approach. Figma
collaborative editor labels elements with floats, then
sorts them numerically.
Thus, any insertion assigns a label inbetween two neighboring 
labels, e.g. `(seq_left + seq_right)/2`.
The literal `/2` might be too risky as it allows for clashes 
in case we skip `src` for efficiency. The particular ratio 
can be randomized, as one of the options.
RDX employs 64 bit integers, not floats, but that changes
nothing substantial.
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
Remember, RDX uses odd revisions for tombstones.


##  The lazy tactic

The lazy tactic is useful when we do not plan to edit an
array much. Lazy elements have no ID at all (all IDs are zeroes).
That way, we save greatly on metadata.
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

##  The roundhouse kick tactic

The above technique hints at an interesting possibility
that is the key feature of DISCONT. 
As you may have noticed, our version of merge sort
processed unsorted inputs in an interesting way.
How far can we generalize this trick? As it seems, pretty
far. The Figma tactic fails when we can no longer
insert any additional elements between the two preexisting
ones. That may happen if their `seqs` differ by 1.
Such a situation is a rare one if we spend the numbering
capacity wisely, but nevertheless possible.

The roundhouse kick tactic is to insert a discontinuity,
a lesser-id element in a way that merge sort processes
the inputs correctly nevertheless.
Essentially, we see an array as a concatenation of several 
monotonously sorted chunks and merge it that way. 
The key here is the handling of "points of discontinuity", 
i.e. locations where the order of IDs is violated, 
`ID_prev >= ID_next`.
Let's call them "cliffs" and "pits" as they look that way 
on a graph. 
Cliff: the last (greatest ID) element of a sorted chunk,
pit: the first one, the lowest ID.
The zero-ID areas we call "plains"; they have discontinuity
in every point. As a very special twist, we count zero
ID as the greatest. That way, we may initialize array
with all-zero-id elements to patch it later.

To guarantee a correct result, our requirement is:
each input must have all its known cliffs and pits present.
Plains can not be abbreviated in any way, obviously.
Additionally, an inserted chunk must stay below its
cliff, i.e. the IDs of inserted elements must be lower 
than the ID of the preexisting element on the left.

The proof of correctness is "left as an exercise for a
reader". In fact, processing of a sorted chunk creates no
intrigue. Merge sort works in the most usual way.

Assuming the inputs contain all the known cliffs
and pits, the tricky part is the handover from one chunk to
another. The key fact is: if the cliff "won" in the heap
ordering and was put into the resulting (merged) array,
its pit will win immediately next, because it has (surprise)
lower ID. Thanks to this fact, chunk handover creates no
intrigue in the case of no concurrent insertions. 

In the case of concurrent insertions into the same location,
things can become quite hairy. For example, elements from
different insertions may start interleaving. To prevent this
effect, inserted chunks can start with a high-ID element.

Whether it is possible to produce a tactic invulnerable to
all such issues is yet to be seen. 
The surprising finding here is that the simplistic O(N)
dense-space-ID-sorting algorithm can be generalized and
non-dense ID spaces (ints) can be used, likely for infinitely
large linear collections.


