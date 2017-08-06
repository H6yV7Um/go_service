<?php
//$client = new Yar_Client('tcp://i.rpc.toutiao.weibo.cn:8080');
$client = new Yar_Client('tcp://localhost:8080');
$client->SetOpt(YAR_OPT_PACKAGER, "msgpack");
$client->setOpt(YAR_OPT_TIMEOUT, 10000);
$client->setOpt(YAR_OPT_CONNECT_TIMEOUT, 10000);
$client->SetOpt(YAR_OPT_PERSISTENT , true);

try {
    print_r($client->__call("Tag.Get", ['uid' => 5711310074, 'imei' => '866646021104255', 'oid' => ['1042015:historyConcept_1422b68e6cd0690a526c0d93071e3260', '1042015:militaryWarHistroy_10000012', '1042015:city_14005108', '1042015:militaryChildType_1000007']]));
} catch (Exception $ex) {
    var_dump($ex);
}
?>
