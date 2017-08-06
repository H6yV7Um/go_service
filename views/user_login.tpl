{{template "header.tpl" .}}
{{template "main_header.tpl" .}}
{{template "left_side.tpl" .}}

<body>
<div class="container">
    <div id="loginbox" class="mainbox col-md-6 col-md-offset-3 col-sm-8 col-sm-offset-2 loginbox">
        <div class="panel panel-info" >
            <div class="panel-heading">
                <div class="panel-title"> 登录 </div>
            </div>
            <div class="panel-body panel-pad">
                <div id="login-alert" class="alert alert-danger col-sm-12 login-alert"></div>
                <form id="loginform" class="form-horizontal" role="form" action="/login" method="post">
                    <div id="loginalert" class="alert alert-danger {{if eq .Error ""}}login-alert{{end}}">
                        <p> 错误: {{.Error}}</p>
                        <span></span>
                    </div>
                    <div class="input-group margT25">
							<span class="input-group-addon">
								<i class="glyphicon glyphicon-user"></i>
							</span>
                        <input id="login-username" type="text" class="form-control" name="username" value="" placeholder="username or email">
                    </div>
                    <div class="input-group margT25">
                        <span class="input-group-addon"><i class="glyphicon glyphicon-lock"></i></span>
                        <input id="login-password" type="password" class="form-control" name="password" placeholder="password">
                    </div>
                    <div class="form-group margT10">
                        <!-- Button -->
                        <div class="col-sm-12 controls">
                            <a id="btn-login" href="#" class="btn btn-block btn-success" onclick="loginform.submit()">登录 </a>
                            <!--input type="submit" value="登录"-->
                        </div>
                    </div>
                </form>
            </div>
        </div>
    </div>
</div>

{{template "footer.tpl" .}}