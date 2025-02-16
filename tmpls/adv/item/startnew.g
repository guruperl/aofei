{{ template "header" .}}
{{ template "itemheader" .}}

<form class="form" method=post action=item>
<input type=hidden name="action" value="insert" />
<input type=hidden name=campaign_id value="{{index .ARGS.campaign_id 0}}" />
<input type=hidden name=campaign_md5 value="{{index .ARGS.campaign_md5 0}}" />
<input type=hidden name=campaign_name value="{{index .ARGS.campaign_name 0}}" />

<h3>新建创意</h3>

<div class="table-responsive">
<table class="table table-striped table-sm">

<tr><td>创意名称:</td><td><input type=text name=item_name size=40 /></td></tr>
<tr><td>起始时间:</td><td><input type=text name=startx size=16 /> 
截止时间:<input type=text name=endx size=16 /></td></tr>

<tr><td>创意尺寸:</td><td><select size=1 name=size_id>
<option value=1>Half Banner 234x60</option>
<option value=2>Banner 468x60</option>
<option value=3 selected>Leaderboard 728x90</option>
<option value=4>Micro Bar 88x31</option>
<option value=5>Button 120x60</option>
<option value=6>Button 120x90</option>
<option value=7>Button 125x125</option>
<option value=8>Vertical Banner 120x240</option>
<option value=9>Skyscraper 120x600</option>
<option value=10>Wide Skyscraper 160x600</option>
<option value=11>Vertical Rectangle 240x400</option>
<option value=12>Small Rectangle 180x150</option>
<option value=13>Small Square 200x200</option>
<option value=14>Square 250x250</option>
<option value=15>3:1 Rectangle 300x100</option>
<option value=16>Medium Rectangle 300x250</option>
<option value=17>Large Rectangle 336x280</option>
<option value=18>Half Page Ad 300x600</option>
</select></td></tr>
<tr><td>结算方式:</td><td>
<input type=radio name=costtype value=CPD><label>CPD</label>
<input type=radio name=costtype value=CPM><label>CPM</label>
<input type=radio name=costtype value=CPC><label>CPC</label>
<input type=radio name=costtype value=CPA><label>CPA</label>
<label> &nbsp; 价格:</label> <input type=text name=cost size=5>
</td></tr>
<tr><td colspan=2> &nbsp; </td></tr>

<tr><td colspan=2> 创意标签定向: </td></tr>
<tr><td>语言:</td><td>
<input type=checkbox name="name=fl_language" value="English" checked />中文
</td></tr>
<tr><td>平台:</td><td>
<input type=checkbox name=fl_platform value="Web" checked /><label>网站</label>
<input type=checkbox name=fl_platform value="Mobile" checked /><label>手机</label>
<input type=checkbox name=fl_platform value="Email" /><label>邮件</label>
<input type=checkbox name=fl_platform value="Video" checked /><label>视频</label>
<input type=checkbox name=fl_platform value="Device" /><label>终端设备</label>
</td></tr>
<tr><td>页面级别:</td><td>
<input type=checkbox name=fl_pagelevel value="Homepage" checked /><label>Homepage</label>
<input type=checkbox name=fl_pagelevel value="Section" checked /><label>Section</label>
<input type=checkbox name=fl_pagelevel value="SubSection" checked /><label>Sub Section</label>
<input type=checkbox name=fl_pagelevel value="Rest" checked /><label>Rest</label>
</td></tr>
<tr><td>时间段:</td><td>
<input type=checkbox name=fl_clock value="0" checked />全部
<input type=checkbox name=fl_clock value="1" checked />1
<input type=checkbox name=fl_clock value="2" checked />2
<input type=checkbox name=fl_clock value="3" checked />3
<input type=checkbox name=fl_clock value="4" checked />4
<input type=checkbox name=fl_clock value="5" checked />5
<input type=checkbox name=fl_clock value="6" checked />6
<input type=checkbox name=fl_clock value="7" checked />7
<input type=checkbox name=fl_clock value="8" checked />8
<input type=checkbox name=fl_clock value="9" checked />9
<input type=checkbox name=fl_clock value="10" checked />10
<input type=checkbox name=fl_clock value="11" checked />11
<input type=checkbox name=fl_clock value="12" checked />12
</td></tr>
<tr><td>空间位置:</td><td>
<input type=checkbox checked name=fl_yaxis value="ScrollUp" />Scroll Up
<input type=checkbox checked name=fl_yaxis value="ScrollDown" />Scroll Down
<input type=checkbox checked name=fl_yaxis value="ScrollMiddle" />Scroll Middle
<input type=checkbox name=fl_yaxis value="Sticky" />Sticky
<input type=checkbox name=fl_yaxis value="PopUnder" />Pop Under
<input type=checkbox checked name=fl_yaxis value="JumpScreen" />Jump Screen
<input type=checkbox name=fl_yaxis value="Rest" checked />Rest
</td></tr>

</table>
<input type=submit value="新建完成" />
</form>

</div>

{{template "footer"}}
