<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
<title>Member Login</title>
</head>

<body>
<table width=100% bgcolor="#ece9d8">
<tr><td>
<A HREF="/">Home</A>
</td><td align=right>
&nbsp;
</td></tr>
</table>

<h3>{{ .Other.Errorstr }}</h3>

<FORM METHOD="POST" ACTION="/goto/adv/e/login">
<INPUT TYPE="HIDDEN" NAME="{{ .Other.Go_uri_name }}" VALUE="{{ .Other.go_uri }}">
<pre>
Email:    <INPUT TYPE="TEXT"     NAME="{{ .Other.Login }}" size=20>
Passowrd: <INPUT TYPE="PASSWORD" NAME="{{ .Other.Password }}" size=20>
<INPUT TYPE="SUBMIT" VALUE=" Log In ">
</pre>
</FORM>

<p>Forgot password?
<a href="/goto/web/e/adv?action=startretrieve">Retrieve password from here</a>.
<a href="/goto/web/e/adv?action=startnew">Apply New Account</a>.
</p>

</body>
</html>
