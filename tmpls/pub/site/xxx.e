[% INCLUDE start.e %]

<h3> <em>New Site</em> </h3>

<form class=niceform method=post action=site>
<input type=hidden name='action' value='insert'>
<fieldset>
<legend>Site Info</legend>
<dl>
	<dt><label for="sitename">Site Name:</label></dt>
	<dd><input type=text name=sitename id=sitename size=40 /></dd>
</dl>
<dl>
	<dt><label for="siteurl">Site URL:</label></dt>
	<dd><input type=text name=siteurl id=siteurl size=40 /></dd>
</dl>
<dl>
	<dt><label for="priority">Priority:</label></dt>
	<dd>
<input type=radio name=priority id=priorityHight value="High"><label>High</label>
<input type=radio name=priority id=priorityStandard value="Standard" checked><label>Standard</label>
<input type=radio name=priority id=priorityLow value="Low"><label>Low</label>
	</dd>
</dl>
</fieldset>
<table>
<tr><td colspan=2> <h3>Site Property</h3> </td></tr>
<tr><td>Language:</td><td><select size=1 name=languageid>
<option value=0>Adjusted</option>
<option value=2>Arabic</option>
<option value=3>Chinese</option>
<option value=1 selected>English</option>
<option value=4>French</option>
<option value=5>Russian</option>
<option value=6>Spanish</option>
</select></td></tr>
<tr><td>Reader Group:</td><td>
<input type=checkbox name=sp_reader value="Men" checked>Men
<input type=checkbox name=sp_reader value="Women" checked>Women
<input type=checkbox name=sp_reader value="Teens">Teens
<input type=checkbox name=sp_reader value="Kids">Kids
</td></tr>
<tr><td>Platform:</td><td>
<input type=radio name=sp_platform value="Web" checked>Web
<input type=radio name=sp_platform value="Mobile">Mobile
<input type=radio name=sp_platform value="Email">Email
<input type=radio name=sp_platform value="Video">Video
<input type=radio name=sp_platform value="Device">Device
</td></tr>
<tr><td>Site Type:</td><td>
<input type=radio name=sp_style value="Content" checked>Content
<input type=radio name=sp_style value="Blog">Blog
<input type=radio name=sp_style value="Directory">Directory
<input type=radio name=sp_style value="Ecommerce">Ecommerce
<input type=radio name=sp_style value="Social">Social Network
<input type=radio name=sp_style value="Search">Search
</td></tr>
<tr><td>Sale Type:</td><td>
<input type=radio name=sp_vertical value="Direct" checked>Direct
<input type=radio name=sp_vertical value="Indirect">Indirect
</td></tr>
[% IF GOTOINSTALL=='adv' %][% INCLUDE startnew_pub.e %][% END %]
[% IF GOTOINSTALL=='pub' %][% INCLUDE startnew_adv.e %][% END %]
<tr><td colspan=2> &nbsp; </td><td>
</table>
<input type=submit value='Add New Site'>
</form>

[% INCLUDE end.e %]
