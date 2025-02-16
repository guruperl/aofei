<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" "http://www.w3.org/TR/html4/loose.dtd">
<html lang="zh">
<head>
<title>Member Login</title>
<meta http-equiv="Content-Type" content="text/html; charset=UTF-8"/>
<style type="text/css">
/*全局样式*/
body{margin: 0;padding: 0;}
img,body,html{border:0;}
address,caption,cite,code,dfn,em,strong,th,var{font-style:normal;font-weight:normal;}
ol,ul{list-style:none;}
caption,th{text-align:left;}
h1,h2,h3,h4,h5,h6{font-size:100%;}
/*边角样式(圆角)*/
.x-box-tl{background:transparent url(images/corners.gif) no-repeat 0 0;zoom:1;}
.x-box-tc{height:8px;background:transparent url(images/tb.gif) repeat-x 0 0;overflow:hidden;}
.x-box-tr{background:transparent url(images/corners.gif) no-repeat right -8px;}
.x-box-ml{background:transparent url(images/l.gif) repeat-y 0;padding-left:4px;overflow:hidden;zoom:1;}
.x-box-mc{background:#eee url(images/tb.gif) repeat-x 0 -16px;padding:4px 10px;font-family:"Myriad Pro","MyriadWeb","Tahoma","Helvetica","Arial",sans-serif;color:#393939;font-size:12px;}
.x-box-mc h3{font-size:14px;font-weight:bold;margin:0 0 4px 0;zoom:1;}
.x-box-mr{background:transparent url(images/r.gif) repeat-y right;padding-right:4px;overflow:hidden;}
.x-box-bl{background:transparent url(images/corners.gif) no-repeat 0 -16px;zoom:1;}
.x-box-bc{background:transparent url(images/tb.gif) repeat-x 0 -8px;height:8px;overflow:hidden;}
.x-box-br{background:transparent url(images/corners.gif) no-repeat right -24px;}
.x-box-tl,.x-box-bl{padding-left:8px;overflow:hidden;}
.x-box-tr,.x-box-br{padding-right:8px;overflow:hidden;}
/*表单样式*/
.loginPanel {
	margin: -140px auto auto -180px;
	position: absolute;
	top: 50%;
	left: 50%;
	height: 400px;
	width:347px
}
.x-form-text {
	height:16px;
	line-height:16px;
	vertical-align:middle;
}
.x-form-text, textarea.x-form-field {
	background:#FFFFFF url(images/text-bg.gif) repeat-x scroll 0pt;
	border:1px solid #B5B8C8;
	padding:1px 1px;
}
/*版权信息*/
.foot{
	font-family:"Myriad Pro","MyriadWeb","Tahoma","Helvetica","Arial",sans-serif;
	color:#aaaaaa;
	font-size:12px;
	text-align:center;
	padding-top:2px;
}
</style>
</head>
<body>

<FORM METHOD="POST" ACTION="/go.fcgi">
<INPUT TYPE="HIDDEN" NAME="[% CREDENTIAL_0 %]" VALUE="[% go_uri %]">
	<div class="loginPanel">
		<div class="x-box-tl">
			<div class="x-box-tr">
				<div class="x-box-tc">
				</div>
			</div>
		</div>

		<div class="x-box-ml">
			<div class="x-box-mr">
				<div class="x-box-mc" style="height: 173px;">
				<img id="j_id2:j_id4" src="images/register.png"/>
					<table id="j_id2:j_id5" cellspacing="3px" style="width:100%">
						<tr>
						<td align="right" colspan="1" rowspan="1" style="padding-right: 3px;">
							<label>Login：</label>
						</td>
						<td colspan="2">
							<label><INPUT TYPE="TEXT"     NAME="[% CREDENTIAL_1 %]" size=20 style="width: 212px;" class="x-form-text"></label>
						</td>
						</tr>
						<tr>
						<td align="right" colspan="1" style="padding-right: 3px;">
							<label>Password：</label>
						</td>
						<td colspan="2">
							<label><INPUT TYPE="PASSWORD" NAME="[% CREDENTIAL_2 %]" size=20 style="width: 212px;" class="x-form-text"></label>
						</td>
						</tr>
												<tr>
						<td align="center" colspan="2">
							&nbsp;</td>
						<td style="padding-right: 20px;">
							<label><INPUT TYPE="SUBMIT" VALUE=" Log In "></label><label style="color:red"></label>
							<label style="color:red">&nbsp;&nbsp;&nbsp;&nbsp;[% errorstr %]</label>
						</td>
						</tr>
						</table>
				</div>
			</div>
		</div>

		<div class="x-box-bl">
			<div class="x-box-br">
				<div class="x-box-bc">
				</div>
			</div>
		</div>

		<div class="foot">CopyRight Chinagoto.com </div>
	</div>
</form>


</body>
</html>

