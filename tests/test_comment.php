<?php
//$client = new Yar_Client('tcp://i.rpc.toutiao.weibo.cn:8080');
$client = new Yar_Client('tcp://localhost:8080');
//$client = new Yar_Client('tcp://172.16.181.234:8081');
$client->SetOpt(YAR_OPT_PACKAGER, "msgpack");
$client->setOpt(YAR_OPT_TIMEOUT, 1000);
$client->setOpt(YAR_OPT_CONNECT_TIMEOUT, 1000);
$client->SetOpt(YAR_OPT_PERSISTENT , true);

try {
    //print_r($client->__call("Service.Status", ['oid' => '']));
    //print_r($client->__call("Comment.Get", ['oid' => '1022:23041848b2fa410102wabb', 'type' => 'hot', 'max'=>366000357, 'count'=>10]));
    //print_r($client->__call("Comment.Get", ['oid' => '1022:23041848b2fa410102wabb', 'max'=>3947626351409721, 'count'=>10]));
    print_r($client->__call("Comment.GetCount", ['oid' => ['2026736001:comos:fxruaee5388342', '2026736001:comos:fxsktkr5729649']]));
    //print_r($client->__call("Comment.Get", ['oid' => '2026736001:comos:fxruaee5388342', 'type' => 'hot']));
} catch (Exception $ex) {
    var_dump($ex);
}
