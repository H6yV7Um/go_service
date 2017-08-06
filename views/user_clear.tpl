{{template "header.tpl" .}}
{{template "main_header.tpl" .}}
{{template "left_side.tpl" .}}

<script language="JavaScript">
    function do_submit() {
        $uid = document.getElementById("uid").value;
        $imei = document.getElementById("imei").value;
        $.get("/rpc?service=User.Clear&Field=channel&Uid=" + $uid + "&Imei=" + $imei, function(data){
            var result = "";
            if(data.Result.Status == 0) {
                result += "清除成功!</br>";
                document.getElementById("result").innerHTML="成功</br>" + data.Result.Data;
            } else {
                result += "清除失败!</br>";
                document.getElementById("result").innerHTML="失败</br>" + data.Result.Error;
            }
            for(i=0; i<data.Data.length; i++) {
                console.log(data.Data[i]);
                result += '<span style="color: #0000ff; ">' + data.Data[i]["field"] + "</span>:" + data.Data[i]["msg"];
            }
            document.getElementById("result").innerHTML = result;
        });
    }

</script>

<!-- Content Wrapper. Contains page content -->
<div class="content-wrapper row">
    <div class="col-md-6 col-md-offset-3">
        <div class="panel panel-info" >
            <div class="panel-heading">
                <div class="panel-title"> 清除用户数据 </div>
            </div>
            <div class="panel-body panel-pad">
                <form id="form1" class="form-horizontal" role="form" method="post">
                    <div class="input-group margT25">
							<span class="input-group-addon">
								<i class="glyphicon glyphicon-user"></i>
							</span>
                        <input id="uid" type="text" class="form-control" name="uid" value="" placeholder="uid">
                    </div>
                    <div class="input-group margT25">
							<span class="input-group-addon">
								<i class="fa fa-list-alt"></i>
							</span>
                        <input id="imei" type="text" class="form-control" name="imei" value="" placeholder="imei">
                    </div>
                    <div class="form-group margT10">
                        <!-- Button -->
                        <div class="col-sm-12 controls">
                            <a id="btn-commit" href="#" class="btn btn-block btn-success" onclick="do_submit();">确定 </a>
                        </div>
                    </div>
                </form>
            </div>
        </div>
    </div>
    <div class="row">
        <div class="col-md-6 col-md-offset-3" id="result">
        </div>
    </div>
</div>
<!-- /.content-wrapper -->

{{template "footer.tpl" .}}