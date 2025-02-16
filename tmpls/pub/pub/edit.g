{{ template "header" .}}
{{ template "pubheader" .}}

<form class="form" action="pub" method=post>
<input type=hidden name=action value="updatepass" />

<h3>修改密码</h3>
<div class="table-responsive">
<table class="table table-striped table-sm">
	<tbody>
<tr>
<td>请输入原密码:</td>
<td><input type=password name=passwd_old size=10 /></td>
</tr>
<tr><td>请输入新密码:</td>
<td><input type=password name=passwd size=10 /></td>
</tr>
<tr>
<td>确认新密码:</td>
<td><input type=password name=confirm size=10 /></td>
</tr>
<tr>
<td colspan=2><input type=submit value="确定修改" /></td>
</tr>
	</tbody>
</table>
</div>

</form>

{{ template "footer" }}
