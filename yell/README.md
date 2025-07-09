#   Lisp

This is a smallish Lisp for scripting and testing the rest of RDX/BRIX.

````
    $ ./lisp 'echo(brix:new({@Alice-123 "Alice":"Had" "Little":"Sheep"}))'
    6972aabe06efd31a57078ed5659ffffaf25cfbd551dd7441f8eca843ef1f930d
    $ ./lisp 'echo(brix:get(6972aa Alice-123))'
    {@Alice-123 "Alice":"Had","Little":"Sheep"}
````
