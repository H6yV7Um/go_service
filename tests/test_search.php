<?php
//$client = new Yar_Client('tcp://172.16.181.234:8080');
$client = new Yar_Client('tcp://i.rpc.toutiao.weibo.cn:8080');
//$client = new Yar_Client('tcp://221.179.175.138:8080');
//$client = new Yar_Client('tcp://localhost:8080');
$client->SetOpt(YAR_OPT_PACKAGER, "msgpack");
$client->setOpt(YAR_OPT_TIMEOUT, 10000);
$client->setOpt(YAR_OPT_CONNECT_TIMEOUT, 10000);
$client->SetOpt(YAR_OPT_PERSISTENT , true);

try {
    for($i=0;$i<1;$i++) {
        print_r($client->__call("Search.Query", ['kw' => ["足球"]]));
    }
} catch (Exception $ex) {
    var_dump($ex);
}
