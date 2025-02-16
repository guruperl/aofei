{{ template "header" .}}
{{ template "advheader" .}}

<form name=form2 class="form" action="adv" method=post>
<input type=hidden name=action value="updatepass" />

<h3>Change Password</h3>
<div class="table-responsive">
<table class="table table-striped table-sm">
	<tbody>
<tr>
<td>Current Password:</td>
<td><input type=password name=passwd_old size=10 /></td>
</tr>
<tr><td>New Password:</td>
<td><input type=password name=passwd size=10 /></td>
</tr>
<tr>
<td>Confirm New:</td>
<td><input type=password name=confirm size=10 /></td>
</tr>
<tr>
<td colspan=2><input type=submit value="Change Now!" /></td>
</tr>
	</tbody>
</table>
</div>

</form>

{{ template "footer" }}
