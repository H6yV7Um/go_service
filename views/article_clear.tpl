{{template "header.tpl" .}}
{{template "main_header.tpl" .}}
{{template "left_side.tpl" .}}

<script language="JavaScript">
    function do_submit() {
        $.get("/rpc?service=Article.Clear", function(data){
            if(data.Result.Status == 0) {
                alert("成功！");
            } else {
                alert("失败：" + data.Result.Error);
            }
        });
    }

</script>

<!-- Content Wrapper. Contains page content -->
<div class="content-wrapper row">
    <div class="col-md-6 col-md-offset-3">
        <div class="panel panel-info" >
            <div class="panel-heading">
                <div class="panel-title"> 清除文章缓存 </div>
                <div class="forgot-password"> ...... </div>
            </div>
            <div class="panel-body panel-pad">
                <form id="form1" class="form-horizontal" role="form" method="post">
                    <div class="input-group margT25">
							<span class="input-group-addon">
								<i class="glyphicon glyphicon-user"></i>
							</span>
                        <input id="Oid" type="text" class="form-control" name="Oid" value="" placeholder="oid">
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
</div>
<!-- /.content-wrapper -->

{{template "footer.tpl" .}}