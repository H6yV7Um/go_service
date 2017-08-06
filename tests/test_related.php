<?php
//$client = new Yar_Client('tcp://172.16.181.234:8080');
$client = new Yar_Client('tcp://s4:8080');
//$client = new Yar_Client('tcp://221.179.175.138:8080');
//$client = new Yar_Client('tcp://localhost:8080');
$client->SetOpt(YAR_OPT_PACKAGER, "msgpack");
$client->setOpt(YAR_OPT_TIMEOUT, 10000);
$client->setOpt(YAR_OPT_CONNECT_TIMEOUT, 10000);
$client->SetOpt(YAR_OPT_PERSISTENT , true);
//$arguments =  array('uid'=>78956, 'oid'=>'3000001725:45a742655333c63b3b4317cc860acfc7', 'tag'=>'1042015:tagCategory_001', 'write_hist'=>true);

try {
    for($i=0;$i<1;$i++) {
        print_r($client->__call("Service.ArticleRelated", ['oid' => '3000000120:7772faebff06cdd70e3db2035390823c', 'uid' => 123]));
    }
} catch (Exception $ex) {
    var_dump($ex);
}
