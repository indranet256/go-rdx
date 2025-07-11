#   RDX Replicated Data eXchange format

This is a reimplementation of [RDX][r] in Go as per Jun 2025.
It supersedes the [2024 Chotki][c] implementation and
uses the same [test set][t] as the [librdx][l] C implementation.

The main change to the previous revision is
that all CRDTs are now implemented by iterator
heap merge, mainly thanks to the new [DISCONT][d]
linear collection CRDT.

##  Changelog

  - 4 Jul 2025: added a small LISP for scripting and testing
    (many thanks to I.Ramadhan)
  - 29 Jun 2025: implemented the basics of the Merklefied LSM 
    format BRIX  

[c]: http://github.com/drpcorg/chotki
[d]: ./DISCOUNT.md
[l]: http://github.com/gritzko/librdx
[r]: ./RDX.md
[t]: ./y.E.md
