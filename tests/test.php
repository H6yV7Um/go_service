<?php
//$client = new Yar_Client('tcp://i.rpc.toutiao.weibo.cn:8080');
$client = new Yar_Client('tcp://localhost:8180');
//$client = new Yar_Client('tcp://221.179.175.138:8080');
//$client = new Yar_Client('tcp://10.73.13.187:8080');
$client->SetOpt(YAR_OPT_PACKAGER, "msgpack");
$client->setOpt(YAR_OPT_TIMEOUT, 10000);
$client->setOpt(YAR_OPT_CONNECT_TIMEOUT, 10000);
$client->SetOpt(YAR_OPT_PERSISTENT , true);
//$arguments =  array('uid'=>78956, 'oid'=>'3000001725:45a742655333c63b3b4317cc860acfc7', 'tag'=>'1042015:tagCategory_001', 'write_hist'=>true);

try {
    //print_r($client->__call("Article.ClearCache", ['oid'=>['2017913001:230461sinanews.sina.cn5e30397e36857a4a']]));
    //print_r($client->__call("Service.Status", ['oid' => '']));
    //for($i=0;$i<1;$i++) {
        print_r($client->__call("Article.MultiFetch", ['oid' => ['1022:2309404040581742507289'], 'fields' => ['famous']]));
    //}
    //$data = $client->__call("Service.ArticleRelated", ['uid'=>78956, 'count'=>10, 'oid'=>'2017913001:230461sinanews.sina.cn5e30397e36857a4a', 'write_hist'=>true]);
    /*for($i=0;$i<2;$i++) {
        print_r($client->__call("Service.ArticleRelated", ['uid'=>2254361222, 'count' => 5, 'oid' => '3000000143:0d46bc3d444c28c0bc3552a55749c94f']));
    }*/
    //print_r($client->__call("Article.Test", ['oid' => ['3000000143:0d46bc3d444c28c0bc3552a55749c94f']]));
    //print_r($client->__call("Service.Status", ['uid'=>0]));
    //print_r($client->__call("User.Get", ["start"=>25, "count"=>5, "uid"=>[2254361222], "imei"=>["359596064197555"], "field"=>["profile", "extra", "favor", "subscribed", "device_type", "device_brand", "device_model", "device_ua"]]));
    //$data = $client->__call("Comment.Get", ['oid'=>'12345']);
    //var_dump($data);
    //$data = $client->__call("Like.Test", ["s" => "alex"]);
    //var_dump($data);
    //$data = $client->__call("Article.Like", ['uid'=>2254361222, 'oid'=>'3000000041:99d6962d27ad05a54b96cba81e5a95fc', 'liked'=>1]);
    //var_dump($data);
    //$data = $client->__call("Article.CheckLike", ['uid'=>2254361222, 'oid'=>['3000000041:99d6962d27ad05a54b96cba81e5a95fc', '1042015:test2']]);
    //print_r($data);
} catch (Exception $ex) {
    var_dump($ex);
}
