<?php
//$client = new Yar_Client('tcp://i.rpc.toutiao.weibo.cn:8080');
$client = new Yar_Client('tcp://10.79.40.80:8001');
//$client = new Yar_Client('tcp://172.16.181.234:8081');
$client->SetOpt(YAR_OPT_PACKAGER, "msgpack");
$client->setOpt(YAR_OPT_TIMEOUT, 1000);
$client->setOpt(YAR_OPT_CONNECT_TIMEOUT, 1000);
$client->SetOpt(YAR_OPT_PERSISTENT , true);

try {
	$params = [
		"uid"=>[10214123123]
	];
    print_r($client->__call("User.GetNotice", $params));
} catch (Exception $ex) {
    var_dump($ex);
}
