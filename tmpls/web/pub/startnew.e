{{ template "header" }}

<form action=pub method=post>
<input type=hidden name=action value="insert" />
<table style="text-align:left">
<tr><td>Email Address：</td><td><input type=text name=email size=30 /></td></tr>
<tr><td>Password：</td><td><input type=password name=passwd size=10 /></td></tr>
<tr><td>Confirm Password：</td><td><input type=password name=confirm size=10 /></td></tr>
<tr><td>Company：</td><td><input type=text name=company size=30 /></td></tr>
<tr><td colspan=2><input type=submit value="Register" /></td></tr>
</table>
</form>

{{ template "footer" }}
