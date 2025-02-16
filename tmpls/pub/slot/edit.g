{{ template "header" .}}
{{ template "slotheader" .}}

<form class="form" action="slot" method=post>
<input type=hidden name="action" value="insert" />
<input type=hidden name="site_id"   value="{{index .ARGS.site_id 0}}" />
<input type=hidden name="site_md5"  value="{{index .ARGS.site_md5 0}}" />
<input type=hidden name="site_name" value="{{index .ARGS.site_name 0}}" />

<h2>修改广告位</h2>
<div class="table-responsive">
<table class="table table-striped table-sm">

<tr><td>广告位名称:</td><td><input type=text name=slot_name size=40></td></tr>
<tr><td>尺寸:</td><td><select size=1 name=size_id>
<option value="1">Leaderboard</option>
<option value="2">Square</option>
<option value="3">Banner</option>
</select></td></tr>

<!--
<tr><td>Language:</td><td>
<input type=radio name="name=qa_language" value="English" />English
<input type=radio name="name=qa_language" value="German" />German
<input type=radio name="name=qa_language" value="French" />French
<input type=radio name="name=qa_language" value="European" />European
<input type=radio name="name=qa_language" value="Chinese" />Chinese
<input type=radio name="name=qa_language" value="Japanese" />Japanese
<input type=radio name="name=qa_language" value="Korean" />Korean
<input type=radio name="name=qa_language" value="Asian" />Asian
<input type=radio name="name=qa_language" value="Arabian" />Arabian
<input type=radio name="name=qa_language" value="Other" selected />Other
</td></tr>-->
<tr><td>类型:</td><td>
<input type=radio name=qa_platform value="Web" checked /><label>网站广告</label>
<input type=radio name=qa_platform value="Mobile" /><label>移动广告</label>
<input type=radio name=qa_platform value="Email" /><label>邮件广告</label>
<!--
<input type=radio name=qa_platform value="Video" /><label>Video</label>
<input type=radio name=qa_platform value="Device" /><label>Device</label>
-->
</td></tr>
<tr><td>广告位所属页面级别:</td><td>
<input type=radio name=qa_pagelevel value="Homepage" /><label>Homepage</label>
<input type=radio name=qa_pagelevel value="Section" /><label>Section</label>
<input type=radio name=qa_pagelevel value="SubSection" /><label>Sub Section</label>
<input type=radio name=qa_pagelevel value="Rest" checked /><label>Rest</label>
</td></tr>
<tr><td>Clock:</td><td>
<input type=radio name=qa_clock value="0" checked />Unknown
<input type=radio name=qa_clock value="1" />1
<input type=radio name=qa_clock value="2" />2
<input type=radio name=qa_clock value="3" />3
<input type=radio name=qa_clock value="4" />4
<input type=radio name=qa_clock value="5" />5
<input type=radio name=qa_clock value="6" />6
<input type=radio name=qa_clock value="7" />7
<input type=radio name=qa_clock value="8" />8
<input type=radio name=qa_clock value="9" />9
<input type=radio name=qa_clock value="10" />10
<input type=radio name=qa_clock value="11" />11
<input type=radio name=qa_clock value="12" />12
</td></tr>
<tr><td>Vertical Location:</td><td>
<input type=radio name=qa_yaxis value="ScrollUp" />Scroll Up
<input type=radio name=qa_yaxis value="ScrollDown" />Scroll Down
<input type=radio name=qa_yaxis value="ScrollMiddle" />Scroll Middle
<input type=radio name=qa_yaxis value="Sticky" />Sticky
<input type=radio name=qa_yaxis value="PopUnder" />Pop Under
<input type=radio name=qa_yaxis value="JumpScreen" />Jump Screen
<input type=radio name=qa_yaxis value="Rest" checked />Rest
</td></tr>

<tr><td colspan=2> &nbsp; </td><td>
</table>
<input type=submit value=" 保存 " />
</form>

</div>

{{ template "footer" .}}
