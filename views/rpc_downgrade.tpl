{{template "header.tpl" .}}
{{template "main_header.tpl" .}}
{{template "left_side.tpl" .}}


<script type="text/javascript">
    $(document).ready(function() {
        $('input[type="checkbox"].flat-green').iCheck({
            checkboxClass: 'icheckbox_flat-green',
            radioClass: 'iradio_flat-green'
        });
        $('input[type="checkbox"]').on('ifChanged', function(event){
            var name = event.target.id;
            var checked = event.target.checked;
            $("#name_param").val(name);
            $("#checked_param").val(checked);
            $("#confirm_box").modal();
        });
    });

    function doCancel() {
        var name = $("#name_param").val();
        $("#" + name).iCheck('toggle');
    }

    function doCheck() {
        var name = $("#name_param").val();
        var checked = $("#checked_param").val();
        var url = "/tools?m=rpc&i=updatedowngrade&name=" + name + "&checked=" + checked;
        console.log(url);
        $.get(url, function(data){
            console.log(data);
            var result = JSON.stringify(data, null, 4);
            console.log(result);
        });
    }
</script>

<div class="modal fade" id="confirm_box" tabindex="-1" role="dialog">
    <div class="modal-dialog">
        <div class="modal-content">
            <div class="modal-header">
                <button type="button" class="close" data-dismiss="modal"><span aria-hidden="true">&times;</span><span class="sr-only">Close</span></button>
                <h4 class="modal-title">确定更改状态?</h4>
            </div>
            <div class="modal-body">
                <h5>如果确认，会实时更改该状态，<span style="color:red">但是不会同步到配置文件</span>，服务重启后会恢复到默认值。</h5>
                <p>如需更新配置文件，请更改代码提交后走正常上线流程。</p>
            </div>
            <div class="modal-footer">
                <input type="hidden" name="name_param" id="name_param"/>
                <input type="hidden" name="checked_param" id="checked_param"/>
                <button type="button" class="btn btn-default" data-dismiss="modal" onclick="doCancel();">关闭</button>
                <button type="button" class="btn btn-primary" data-dismiss="modal" onclick="doCheck();">确认</button>
            </div>
        </div>
    </div>
</div>

<!-- Content Wrapper. Contains page content -->
<div class="content-wrapper row">
    <div class="col-xs-10" style="padding: 10px">
        <div class="panel panel-info" style="padding: 10px">
            {{range $i, $switch := .Switches}}
            <div class="form-group">
                <label>
                    <input name="{{$switch.Name}}" id="{{$switch.Name}}" type="checkbox" class="flat-green" {{$switch.Checked}} {{$switch.Disabled}}>
                    {{$switch.Name}}<br/>
                    <small style="padding-left: 20px">{{$switch.Description}}</small>
                </label>
            </div>
            {{end}}
        </div>
    </div>
</div>

<!-- /.content-wrapper -->

{{template "footer.tpl" .}}