<?php
//$client = new Yar_Client('tcp://i.rpc.toutiao.weibo.cn:8080');
$client = new Yar_Client('tcp://localhost:8080');
//$client = new Yar_Client('tcp://s3:8081');
$client->SetOpt(YAR_OPT_PACKAGER, "msgpack");
$client->setOpt(YAR_OPT_TIMEOUT, 10000);
$client->setOpt(YAR_OPT_CONNECT_TIMEOUT, 10000);
$client->SetOpt(YAR_OPT_PERSISTENT , true);

try {
    $uid = 2254361222;
    $imei = "";
    //print_r($client->__call("User.AddSubscribe", ['uid' => $uid, 'target_type' => 'tag', 'target_id' =>
    //    ['1042015:moviePerson_314859', '1042015:abilityTag_10457', '1042015:foodMenu_3e1870d5583681f0','1042015:moviePerson_554397', '1042015:movie_177319']]));
    //print_r($client->__call("User.DelSubscribe", ['uid' => $uid, 'target_type' => 'tag', 'target_id' => ['1042015:moviePerson_314859']]));
    //print_r($client->__call("User.DelSubscribe", ['uid' => $uid, 'target_type' => 'tag', 'target_id' => ['1042015:abilityTag_10457']]));
    /*print_r($client->__call("User.Update", ['uid' => $uid, 'imei' => $imei, 'values' => [
        'time' => time(),
        'updtime' => time() + 100,
        'token' => 'test_token',
        'wm' => '10000_10011',
        'aid' => 'test_aid3',
        'device_brand' => 'test_brand'
    ]]));*/
    print_r($client->__call("User.Get", ['uid' => [$uid], 'imei' => [$imei], 'field' => ['favor'], 'count' => 20]));
    //print_r($client->__call("User.Get", ['uid' => [$uid], 'imei' => [$imei], 'field' => ['favor','subscribed','profile','extra']]));
} catch (Exception $ex) {
    var_dump($ex);
}
