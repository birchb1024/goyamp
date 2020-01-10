#!/bin/bash
#
# Execute goyamp on a file and display the output in a second column.
# Useful for demonstrating or testing simple expansions.
#
# Usage example:
#$ ./side-by-side.sh examples/repeat-map.yaml
#
##----Input--------examples/repeat-map.yaml------------------                             | #----Output-------------------------
#define:                                                                                  | ---
#  fudge:                                                                                 | - key = a, value = 1
#    a: 1                                                                                 | - key = b, value = 2
#    b: 2                                                                                 | - key = c, value = [ 1, 2 ]
#    c: [1,2]                                                                             | - 'key = d, value = { d1 : fu , d2 : bar  }'
#    d:                                                                                   | 
#      d1: "fu"                                                                           | 
#      d2: bar                                                                            | 
#---                                                                                      | 
#repeat:                                                                                  | 
#   for: $key                                                                             | 
#   in: fudge                                                                             | 
#   body:                                                                                 | 
#       key = {{ $key }}, value = {{ fudge.$key }}                                        | 
#
#
set -e
set -u


export PATH=../:$PATH
echo "#----Input--------$@------------------" >/tmp/$$_in.yaml
echo "#----Output-------------------------" >/tmp/$$_out.yaml
cat $@ >> /tmp/$$_in.yaml
cat $@ | goyamp - 2>&1 >>/tmp/$$_out.yaml
pr -m -t -W 180 -S" | " /tmp/$$_in.yaml /tmp/$$_out.yaml
rm /tmp/$$_*.yaml

