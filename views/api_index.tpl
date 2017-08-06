{{template "header.tpl" .}}
{{template "main_header.tpl" .}}
{{template "left_side.tpl" .}}


<script type="text/javascript">
    function updateMethod() {
        var service = $("#service").val();
        var methods = mList[service];
        $("#method").empty();
        $.each(methods, function (i, item) {
            $("#method").append("<option value='" + item.Name + "'>&nbsp;" + item.Name + "</option>");
        });
        $("#method").trigger("change");
    }

    function getArgs() {
        var service = $("#service").val();
        var method = $("#method").val();

        for(i=0; i<mList[service].length; i++) {
            if(mList[service][i].Name == method) {
                return mList[service][i].Args;
            }
        }
        return undefined
    }

    function updateArgs() {
        var service = $("#service").val();
        var method = $("#method").val();
        var args = getArgs();
        if(args == undefined) {
            return;
        }
        var args_html = "";
        for(i=0; i<args.length; i++) {
            args_html += "<div class=\"row\"><label for=\"method_name\">" + args[i].Name +
                    " <span style=\"font-weight: 100;font-size:12px;\">(" + args[i].Type + ", " + args[i].Must + ", " + args[i].Comment + ")</span>" +
                    "<input class=\"js-states form-control\" size=\"" + args[i].Size + "\" id=\"" + args[i].Name + "\" onchanged=\"onArgsChanged()\"/></div>";
        }
        $("#args").html(args_html);
    }

    function onArgsChanged() {
        console.log("args changed");
    }

    function doSubmit() {
        var method = $("#method").val();
        var api_url = "v.top.weibo.cn/2/" + method;

        var args = getArgs();
        if(args == undefined) {
            alert("获取参数出错!");
            return;
        }
        for(var i=0; i<args.length; i++) {
            var arg = $("#" + args[i].Name).val();
            if(i==0) {
                api_url += "?" + args[i].Name + "=" + arg;
            } else {
                api_url += "&" + args[i].Name + "=" + arg;
            }
        }
        var url = "http://" + window.location.host + "/tools?m=api&i=proxy&url=" + escape(api_url);

        $("#request_url").val(url);
        $.get(url, function(data){
            var result = JSON.stringify(data, null, 4);
            console.log(result);
            $("#result").html("<pre><code class=\"language-json\">" + result + "</code></pre>");
            $('pre code').each(function(i, block) {
                hljs.highlightBlock(block);
            });
        });
    }

    var mList = eval("("+ {{.mListJson}} +")");
    $(document).ready(function() {
        $(".service").select2();
        $("#service").on("change", function (e) {
            updateMethod();
        });

        $(".method").select2();
        $("#method").on("change", function (e) {
            updateArgs();
        });

        updateMethod();
    });
</script>


<!-- Content Wrapper. Contains page content -->
<div class="content-wrapper row">
    <div class="col-xs-10 col-xs-offset-1">
        <div class="panel panel-info" >
            <div class="panel-heading">
                <div class="panel-title"> Api接口测试（未完成）<span class="pull-right">
                        <a class="text-success" target="_blank" href="http://wiki.intra.sina.com.cn/pages/viewpage.action?pageId=26741058">接口文档</a></span></div>
            </div>
            <div class="panel-body panel-pad">
                <form id="form1" class="form-horizontal" role="form" method="post">
                    <div class="row">
                    <div class="col-xs-5 col-xs-offset-1">
                    <label for="service">选择接口分组
                        <select class="js-states form-control" id="service" name="service">
                            {{range $service, $method := .mList}}
                            <option value="{{$service}}">{{$service}}</option>
                            {{end}}
                        </select>
                    </label>
                    </div>
                        <div class="col-xs-5">
                    <label for="method">选择接口URI
                        <select class="js-states form-control" id="method" name="method">
                        </select>
                    </label>
                    </div>
                    </div>

                    <div class="row well well-sm col-xs-8 col-xs-offset-1" style="margin-top:15px;">
                        <span style="font-weight: 500;">参数</span>
                            <div id="args" style="padding:15px;"></div>
                    </div>

                    <div class="row">
                    <div class="row col-xs-10 col-xs-offset-1">
                        <label for="request_url">请求URL
                            <input class="js-states form-control" id="request_url" size="100%"/>
                    </div>
                    </div>

                    <div class="form-group margT10">
                        <div class="col-xs-3 col-xs-offset-1 controls">
                            <a id="btn-commit" href="#" class="btn btn-block btn-success" onclick="doSubmit();">提交 </a>
                        </div>
                    </div>

                </form>
            </div>

            <div class="row">
                <div class="col-xs-10 col-md-offset-1" id="result">
                </div>
            </div>

        </div>
    </div>
</div>
<!-- /.content-wrapper -->

{{template "footer.tpl" .}}