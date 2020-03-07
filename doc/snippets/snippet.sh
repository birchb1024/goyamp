#!/bin/bash
export PATH=../../:$PATH
echo "#----Input--------------------------" >/tmp/$$_in.yaml
echo "#----Output-------------------------" >/tmp/$$_out.yaml
cat $1 >> /tmp/$$_in.yaml
goyamp $1 >>/tmp/$$_out.yaml
pr -m -t -W 120 -S" | " /tmp/$$_in.yaml /tmp/$$_out.yaml
rm /tmp/$$_*.yaml

