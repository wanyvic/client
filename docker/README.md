
```bash
### build 
./build.sh -t huobi/btcproxy -d Release 

#or
sudo docker build -t huobi/btcproxy -f Dockerfile --build-arg BUILD_TYPE=Release --build-arg=$(nproc) --build-arg GIT_DESCRIBE=$(git describe --tag --long) --build-arg APT_MIRROR_URL=http://cn-north-1b.clouds.archive.ubuntu.com/ubuntu ..

### run
sudo docker run --name=bsv-btcproxy --restart=always -v=$CONFIGPATH/run_btcproxy:/work/config -d huobi/btcproxy btcagent -c /work/config/agent_config.json -l /work/config/log_btcagent
```