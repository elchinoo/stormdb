HammerDB - PGE v17 - 3 (4, 8G)
p-yjm8281xi6-rw-external-c93d591d0e53e84f.elb.us-east-1.amazonaws.com

IMDB - Community v17 - 1 (4, 8GB)
p-7cn6bhuvol-rw-external-7cc6b3b8a35397dc.elb.us-east-1.amazonaws.com

IMDB - Community v17 - 1x3 (4, 8GB)
p-qw3l29elvw-rw-external-7ac1b178a9521709.elb.us-east-1.amazonaws.com


IMDB - PGE v17 - 1 (4, 8G)
p-sx1uhyb5ku-rw-external-7871e1cc7c70f89a.elb.us-east-1.amazonaws.com

IMDB - PGE v17 - 1x3 (4, 8G)
p-madcfeegq9-rw-external-16a88bc06d728d9d.elb.us-east-1.amazonaws.com

PGSTORM AI - PGE v17 - 1 (8, 16G)
p-27od0xx18c-rw-external-b25bf40fc3adde43.elb.us-east-1.amazonaws.com



IMDB - EPAS v17 - 1x3 (4,8G)
p-5z2mcdtt4g-rw-external-4742739c702557c6.elb.us-east-1.amazonaws.com

IMDB - EPAS v16 - 1x3 (4, 8G)
p-rzjaqzb0yn-rw-external-31683a08ddd89515.elb.us-east-1.amazonaws.com

Could you help me to generate the configuration files to run ecommerce_mixed, imdb_mixed, tpcc, and realworld in below boxes? Please salve the files in the folder hcp because those are local tests and I don't want them to go to github:

Host name: IMDB - EPAS v17 - 1x3 (4,8G)
Host address: p-5z2mcdtt4g-rw-external-4742739c702557c6.elb.us-east-1.amazonaws.com
Host name: IMDB - EPAS v16 - 1x3 (4, 8G)
Host address: p-rzjaqzb0yn-rw-external-31683a08ddd89515.elb.us-east-1.amazonaws.com


hcp/config_ecommerce_mixed_epas16.yaml
hcp/config_ecommerce_mixed_epas17.yaml
hcp/config_imdb_mixed_epas16.yaml
hcp/config_imdb_mixed_epas17.yaml
hcp/config_realworld_epas16.yaml
hcp/config_realworld_epas17.yaml
hcp/config_tpcc_epas16.yaml
hcp/config_tpcc_epas17.yaml

We can update the credentials in the folder hcp as it isn't git logged
username: "edb_admin"
password: "mattdemo123!"
dbname: "edb_admin"


Could you also help me to generate the configuration files to run ecommerce_mixed, imdb_mixed, tpcc, and realworld in below boxes? Please salve the files in the folder hcp because those are local tests and I don't want them to go to github:

Host name: STORM - PGE v17 - 1x3 (4, 8GB)
Host address: p-kgd54g3gg7-rw-external-2e5e606c35f06e33.elb.us-east-1.amazonaws.com
Host name: STORM - PGE v16 - 1x3 (4, 8GB)
Host address: p-ys0nl9245c-rw-external-d6e5d894e2a130a6.elb.us-east-1.amazonaws.com

username: "edb_admin"
password: "mattdemo123!"
dbname: "edb_admin"


Can you help me to update below script to go through all the configurations for the epas17 database and run all the loads for the loads in the example? For example, run the ecommerce_mixed for 16, 36, 64, and 128 workers? Please save it as the "servername.sh", for example epas17.sh, and do it for all the loads in the hcp folder:

#!/bin/bash
# progressive_load.sh - Gradually increase load

WORKERS=(16 36 64 128)
for w in "${WORKERS[@]}"; do
    echo "Testing with $w workers"
    ./stormdb -c hcp/config_ecommerce_mixed_epas17.yaml -workers=$w -duration=60m > "results_${w}workers.log"
    
    # Analysis pause
    sleep 60
done


max_connections: 300
checkpoint_timeout: 30min