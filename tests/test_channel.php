<?php
//$client = new Yar_Client('tcp://i.rpc.toutiao.weibo.cn:8080');
$client = new Yar_Client('tcp://localhost:8080');
$client->SetOpt(YAR_OPT_PACKAGER, "msgpack");
$client->setOpt(YAR_OPT_TIMEOUT, 10000);
$client->setOpt(YAR_OPT_CONNECT_TIMEOUT, 10000);
$client->SetOpt(YAR_OPT_PERSISTENT , true);

try {
    //$uid = 2254361222;
    $uid = 0;
    $imei = "869580029461853";
    /*print_r($client->__call("User.UpdateChannel", ['uid' => $uid, 'data' => [
        'attributes' => ["is_autoadd" => 1, "subscribed_udtime" => 1463067669],
        'subscribed' => [
            ["cate_id" => "0"],
            ["cate_id" => "1"],
            ["cate_id" => "14"],
            ["cate_id" => "0"],
            ["cate_id" => "1042015:province_11"],
            ["cate_id" => "1001"],
            ["cate_id" => "1002"]
        ]
    ]]));*/
    //print_r($client->__call("User.DelChannel", ['uid' => $uid, 'target_type' => 'tag', 'target_id' => ['1042015:moviePerson_314859']]));
    $result = $client->__call("User.GetChannel", ['uid' => $uid, 'imei' => $imei]);
    print_r($result);
    //print_r($client->__call("Tag.Get", ['uid'=>$uid, 'imei'=>'x', 'oid'=>['1042015:abilityTag_10457', '1042015:moviePerson_314859']]));
} catch (Exception $ex) {
    var_dump($ex);
}
