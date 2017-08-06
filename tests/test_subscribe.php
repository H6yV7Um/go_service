<?php
$client = new Yar_Client('tcp://i.rpc.toutiao.weibo.cn:8080');
//$client = new Yar_Client('tcp://localhost:8080');
$client->SetOpt(YAR_OPT_PACKAGER, "msgpack");
$client->setOpt(YAR_OPT_TIMEOUT, 10000);
$client->setOpt(YAR_OPT_CONNECT_TIMEOUT, 10000);
$client->SetOpt(YAR_OPT_PERSISTENT , true);

try {
    $uid = 2254361222;
    $imei = "";
    //print_r($client->__call("User.AddSubscribe", ['uid' => $uid, 'target_type' => 'tag', 'target_id' =>
    //    ['1042015:foodMenu_5dae8198d4baa74f', '1042015:abilityTag_10457', '1042015:foodMenu_3e1870d5583681f0','1042015:moviePerson_554397', '1042015:movie_177319']]));
    //print_r($client->__call("User.DelSubscribe", ['uid' => $uid, 'target_type' => 'tag', 'target_id' => ['1042015:moviePerson_314859']]));
    //print_r($client->__call("User.DelSubscribe", ['uid' => $uid, 'target_type' => 'tag', 'target_id' => ['1042015:abilityTag_10457']]));
    print_r($client->__call("User.GetSubscribe", ['uid' => $uid, 'imei' => $imei, 'target_type' => 'tag', 'start' => 0, 'count' => 10, 'target_count' => 5]));
    //print_r($client->__call("Tag.Get", ['uid'=>$uid, 'imei'=>'x', 'oid'=>['1042015:abilityTag_10457', '1042015:moviePerson_314859']]));
} catch (Exception $ex) {
    var_dump($ex);
}
