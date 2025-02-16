[% INCLUDE start.e %]

[% SET item=edit.0 %]

<div class="ui-layout-west">
        <ul>
        <li><em>[% item.sitename %]</em>
          <ul>
            <li>[% IF item.accessorder=='Inherit' %]<a href="ac?action=topics&entitytype=siteid&accessorder=[% item.accessorder %]&siteid=[% item.siteid %]&sitemd5=[% item.sitemd5 %]&sitename_esc=[% item.sitename_esc %]" onClick="return (confirm('The access control of this site inherits your account setting; do you want to create its own control?')) ? true : false;">New Access Control</a>[% ELSE %]<a href="ac?action=topics&entitytype=siteid&accessorder=[% item.accessorder %]&siteid=[% item.siteid %]&sitemd5=[% item.sitemd5 %]&sitename_esc=[% item.sitename_esc %]">Edit Access Control</a>[% END %] <p></p></li>

            <li><a href="slot?action=startnew&siteid=[% item.siteid %]&sitemd5=[% item.sitemd5 %]&sitename_esc=[% item.sitename_esc %]">Create New Slot</a></li>
			<li><em>Existing Slots</em>:
			<ul>
            [% FOREACH follow IN item.Goto_Slot_Model_topics %][% IF follow.status=='Yes' %]<li><a href="slot?action=edit&slotid=[% follow.slotid %]&siteid=[% siteid %]&sitemd5=[% item.sitemd5 %]&sitename_esc=[% item.sitename_esc %]">[% follow.slotname %]</a></li>[% END %][% END %]
			</ul> <p></p></li>

            <li><a href="page?action=edit&pageid=0&siteid=[% item.siteid %]&sitemd5=[% item.sitemd5 %]&sitename_esc=[% item.sitename_esc %]">Create New Page</a></li>
			<li><em>Existing Pages</em>:
			<ul>
            [% FOREACH follow IN item.Goto_Page_Model_topics %]<li><a href="page?action=edit&pageid=[% follow.pageid %]&siteid=[% siteid %]&sitemd5=[% item.sitemd5 %]&sitename_esc=[% item.sitename_esc %]">[% follow.pagename %]</a></li>[% END %]
			</ul></li>

        </ul></li>
</ul>
</div>
<div class="ui-layout-center">

<div class="siteEdit">
<h3>[% item.sitename %]</h3>

<form name=site method=post action=site>
<input id="action" name="action" type="hidden" value="update" />
<input type=hidden name='siteid' value='[% item.siteid %]'>
<table border=0>
<tr><td>Site Name:</td><td><input type=text value='[% item.sitename %]' name=sitename size=40></td></tr>
<tr><td>URL:</td><td><input type=text value='[% item.siteurl %]' name=siteurl size=40></td></tr>
<tr><td>Priority:</td><td>
<input type=radio [% IF item.priority=='High' %]checked[% END %] name=priority value="High"><label>High</label>
<input type=radio [% IF item.priority=='Standard' %]checked[% END %] name=priority value="Standard"><label>Standard</label>
<input type=radio [% IF item.priority=='Low' %]checked[% END %] name=priority value="Low"><label>Low</label>
</td></tr>
<tr><td colspan=2> <h3>Site Property</h3> </td></tr>
<tr><td>Language:</td><td><select size=1 name=languageid>
<option [% IF item.languageid==0 %]selected[% END %] value=0>Adjusted</option>
<option [% IF item.languageid==2 %]selected[% END %] value=2>Arabic</option>
<option [% IF item.languageid==3 %]selected[% END %] value=3>Chinese</option>
<option [% IF item.languageid==1 %]selected[% END %] value=1>English</option>
<option [% IF item.languageid==4 %]selected[% END %] value=4>French</option>
<option [% IF item.languageid==5 %]selected[% END %] value=5>Russian</option>
<option [% IF item.languageid==6 %]selected[% END %] value=6>Spanish</option>
</select></td></tr>
<tr><td>Reader Group:</td><td>
<input type=checkbox [% IF item.sp_r_men==1 %]checked[% END %] name=sp_reader value="Men" checked><label>Men</label>
<input type=checkbox [% IF item.sp_r_women==1 %]checked[% END %] name=sp_reader value="Women" checked><label>Women</label>
<input type=checkbox [% IF item.sp_r_teens==1 %]checked[% END %] name=sp_reader value="Teens"><label>Teens</label>
<input type=checkbox [% IF item.sp_r_kids==1 %]checked[% END %] name=sp_reader value="Kids"><label>Kids</label>
</td></tr>
<tr><td>Platform:</td><td>
<input type=radio [% IF item.sp_platform=="Web" %]checked[% END %] name=sp_platform value="Web" checked><label>Web</label>
<input type=radio [% IF item.sp_platform=="Mobile" %]checked[% END %] name=sp_platform value="Mobile"><label>Mobile</label>
<input type=radio [% IF item.sp_platform=="Email" %]checked[% END %] name=sp_platform value="Email"><label>Email</label>
<input type=radio [% IF item.sp_platform=="Video" %]checked[% END %] name=sp_platform value="Video"><label>Video</label>
<input type=radio [% IF item.sp_platform=="Device" %]checked[% END %] name=sp_platform value="Device"><label>Device</label>
</td></tr>
<tr><td>Site Type:</td><td>
<input type=radio name=sp_style [% IF item.sp_style=="Content" %]checked[% END %] value="Content"><label>Content</label>
<input type=radio name=sp_style [% IF item.sp_style=="Blog" %]checked[% END %] value="Blog"><label>Blog</label>
<input type=radio name=sp_style [% IF item.sp_style=="Directory" %]checked[% END %] value="Directory"><label>Directory</label>
<input type=radio name=sp_style [% IF item.sp_style=="Ecommerce" %]checked[% END %] value="Ecommerce"><label>Ecommerce</label>
<input type=radio name=sp_style [% IF item.sp_style=="Social" %]checked[% END %] value="Social"><label>Social Network</label>
<input type=radio name=sp_style [% IF item.sp_style=="Search" %]checked[% END %] value="Search"><label>Search</label>
</td></tr>
<tr><td>Sale Type:</td><td>
<input type=radio name=sp_vertical [% IF item.sp_vertical=="Direct" %]checked[% END %] value="Direct"><label>Direct</label>
<input type=radio name=sp_vertical [% IF item.sp_vertical=="Indirect" %]checked[% END %] value="Indirect"><label>Indirect</label>
</td></tr>
[% IF GOTOINSTALL=='adv' %][% INCLUDE edit_pub.e %][% END %]
[% IF GOTOINSTALL=='pub' %][% INCLUDE edit_adv.e %][% END %]
<tr><td colspan=2> &nbsp; </td><td>
</table>
<input type=submit value='Update Site'>
</form>
</div>
<br />
<table border="0">
<tr>
	<td valign="top">
		<h3 class="curTitle">My Channels</h3>
		<form name=chbelong method=post action=chbelong>
		<input type=hidden name='action' value='update'>
		<input type=hidden name='entitytype' value='siteid'>
		<input type=hidden name='siteid' value='[% item.siteid %]'>
		<input type=hidden name='sitemd5' value='[% item.sitemd5 %]'>
		<input type=hidden name='sitename' value='[% item.sitename  %]'>
		
		<pre>[% FOREACH ch IN CHANNELS %]
		<input [% IFED('checked', ch.channelid, item.Goto_Chbelong_Model_topics, 'channelid') %] type=checkbox name=channelid value="[% ch.channelid %]">[% ch.channelname %][% END %]
		</pre>
		
		<input type=submit value='Update Channels'>
		</form>
	</td>
	<td width=10>
	</td>
	<td valign="top">
		<h3 class="curTitle">Campaign Channel Rules</h3>
		<form name=chac method=post action=chac>
		<input type=hidden name='action' value='update'>
		<input type=hidden name='entitytype' value='siteid'>
		<input type=hidden name='siteid' value='[% item.siteid %]'>
		<input type=hidden name='sitemd5' value='[% item.sitemd5 %]'>
		<input type=hidden name='sitename' value='[% item.sitename %]'>
		
		<label>By default, all cmpaign channels will be:</label> &nbsp; &nbsp;
		<input type=radio [% IF item.channelorder=='AllowDeny' %]checked[% END %] name=channelorder value='AllowDeny' onclick="document.getElementById('goto_ac').innerHTML='I allow only the following:'"><label>denied</label> &nbsp;
		<input type=radio [% IF item.channelorder=='DenyAllow' %]checked[% END %] name=channelorder value='DenyAllow' onclick="document.getElementById('goto_ac').innerHTML='I deny only the following:'"><label>allowed</label>
		<p></p>
		<div id="goto_ac">I [% IF item.channelorder=='AllowDeny' %]allow[% ELSE %]deny[% END %] only the following:</div>
		
		<pre>[% FOREACH ch IN CHANNELS %]
		<input [% IFED('checked', ch.channelid, item.Goto_Chac_Model_topics, 'channelid') %] type=checkbox name=channelid value="[% ch.channelid %]">[% ch.channelname %][% END %]
		</pre>
		
		<input type=submit value='Update Matchings'>
		
		</form>
	</td>
</tr>
</table>

</div>
[% INCLUDE end.e %]
