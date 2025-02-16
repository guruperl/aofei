[% INCLUDE start.e %]

[% SET item=edit.0 %]

<div class="ui-layout-west">
<ul id="treeList">
        <li><a href="campaign?action=edit&campaignid=[% campaignid %]">[% campaignname %]</a>
			<p></p>
            <ul>
            <li><em>[% item.itemname %]</em>
			    <ul>

				<li><a href="targetname?action=startnew&target=[% item.target %]&entitytype=item&itemid=[% item.itemid %]&itemmd5=[% item.itemmd5 %]&itemname_esc=[% item.itemname_esc %]&campaignid=[% campaignid %]&campaignmd5=[% campaignmd5 %]&campaignname_esc=[% campaignname_esc %]" [% IF item.target=='Inherit' %]onClick="return (confirm('The targeting of this item inherits your campaign setting; do you want to create its own targeting?')) ? true : false;"[% END %]>Individual Targeting</a></li>
			</ul></li>
        </ul></li>
</ul>
</div>

<div class="ui-layout-center">

<form method=post action=item >
<input type=hidden name='action' value='update'>
<input type=hidden name='itemid' value='[% item.itemid %]'>
<input type=hidden name='campaignid' value='[% campaignid %]'>
<input type=hidden name='campaignmd5' value='[% campaignmd5 %]'>
<input type=hidden name='campaignname' value="[% campaignname %]">

<table border=0>
<tr><td>Lineitem Name:</td><td><input type=text name=itemname value='[% item.itemname %]' size=40></td></tr>
<tr><td>Size:</td><td><select size=1 name=sizeid>
<option [% IF item.sizeid==1 %]selected[% END %] value=1>Half Banner 234x60</option>
<option [% IF item.sizeid==2 %]selected[% END %] value=2>Banner 468x60</option>
<option [% IF item.sizeid==3 %]selected[% END %] value=3>Leaderboard 728x90</option>
<option [% IF item.sizeid==4 %]selected[% END %] value=4>Micro Bar 88x31</option>
<option [% IF item.sizeid==5 %]selected[% END %] value=5>Button 120x60</option>
<option [% IF item.sizeid==6 %]selected[% END %] value=6>Button 120x90</option>
<option [% IF item.sizeid==7 %]selected[% END %] value=7>Button 125x125</option>
<option [% IF item.sizeid==8 %]selected[% END %] value=8>Vertical Banner 120x240</option>
<option [% IF item.sizeid==9 %]selected[% END %] value=9>Skyscraper 120x600</option>
<option [% IF item.sizeid==10 %]selected[% END %] value=10>Wide Skyscraper 160x600</option>
<option [% IF item.sizeid==11 %]selected[% END %] value=11>Vertical Rectangle 240x400</option>
<option [% IF item.sizeid==12 %]selected[% END %] value=12>Small Rectangle 180x150</option>
<option [% IF item.sizeid==13 %]selected[% END %] value=13>Small Square 200x200</option>
<option [% IF item.sizeid==14 %]selected[% END %] value=14>Square 250x250</option>
<option [% IF item.sizeid==15 %]selected[% END %] value=15>3:1 Rectangle 300x100</option>
<option [% IF item.sizeid==16 %]selected[% END %] value=16>Medium Rectangle 300x250</option>
<option [% IF item.sizeid==17 %]selected[% END %] value=17>Large Rectangle 336x280</option>
<option [% IF item.sizeid==18 %]selected[% END %] value=18>Half Page Ad 300x600</option>
</select></td></tr>
<tr><td>Cost Type:</td><td>
<input [% IF item.costtype=='CPD' %]checked[% END %] type=radio name=costtype value=CPD><label>CPD</label>
<input [% IF item.costtype=='CPM' %]checked[% END %] type=radio name=costtype value=CPM><label>CPM</label>
<input [% IF item.costtype=='CPC' %]checked[% END %] type=radio name=costtype value=CPC><label>CPC</label>
<input [% IF item.costtype=='CPA' %]checked[% END %] type=radio name=costtype value=CPA><label>CPA</label>
&nbsp;
<label> &nbsp; Price:</label> <input type=text name=cost value='[% item.cost %]' size=5>
</td></tr>
<tr><td>Quantity:</td><td><input type=text name=quantity value='[% item.quantity %]' size=9>
<label> &nbsp; Delivery Rate:</label> <select size=1 name=deliveryrate>
<option [% IF item.deliveryrate=='Fast' %]selected[% END %] value='Fast'>Fast</option>
<option [% IF item.deliveryrate=='Even' %]selected[% END %] value='Even'>Even</option>
</select></td></tr>
<tr><td valign=top>Frequency Caps:</td><td>
<table>
<tr><th>Type</th><th>Number</th><th>Period</th><th>Throttle</th></tr>
<tr><td>Impression</td>
<td><input type=text value='[% item.cpm_fc %]' name=cpm_fc size=3></td>
<td><input type=text value='[% item.cpm_length %]' name=cpm_length size=9></td>
<td><input type=text value='[% item.cpm_throttle %]' name=cpm_throttle size=9></td></tr>
<tr><td>Clicks</td>
<td><input type=text value='[% item.cpc_fc %]' name=cpc_fc size=3></td>
<td><input type=text value='[% item.cpc_length %]' name=cpc_length size=9></td>
<td></td></tr>
<tr><td>Actions</td>
<td><input type=text value='[% item.cpa_fc %]' name=cpa_fc size=3></td>
<td><input type=text value='[% item.cpa_length %]' name=cpa_length size=9></td>
<td></td></tr>
</table>
</td></tr>
<tr><td>Page Cap: </td><td><input type=text name=pagecap size=1 value='[% item.pagecap %]'></td></tr>
[% INCLUDE edit_item.e %]
<tr><td colspan=2> &nbsp; </td></tr>
</table>
<input type=submit value='Update Lineitem'>
</form>

<script type="text/javascript">
$(function() {
  $("#creativeremove").click(function(){
	var f = document.forms['creative'];
    var obj = $("input:radio[name='creativeid']:checked");
    if (obj == undefined) return false;

	var data = "action=delete&creativeid="+obj.val()+"&itemid="+f.elements["itemid"].value+"&itemmd5="+f.elements["itemmd5"].value+"&campaignid="+f.elements["campaignid"].value+"&campaignmd5="+f.elements["campaignmd5"].value;
$.ajax({
		type: "POST",
		url: "../t/creative",
		data: data,
		success: function(msg){
			if (msg=="0" || msg=="0\n") {
				obj.closest('tr').remove();
			} else {
				alert( msg );
			}
		}
	});
  });
  $("#creativesubmit").click(function(){
	var f = document.forms['creative'];
    var content= escape(f.elements['content'].value);
    var creativename= escape(f.elements['creativename'].value);

	var data = "action=insert&content="+content+"&creativename="+creativename+"&itemid="+f.elements["itemid"].value+"&itemmd5="+f.elements["itemmd5"].value+"&campaignid="+f.elements["campaignid"].value+"&campaignmd5="+f.elements["campaignmd5"].value;
$.ajax({
		type: "POST",
		url: "../t/creative",
		data: data,
		success: function(msg){
			var re = /^\d+/;
			var found = msg.match(re);
			if (found!=undefined) {
				$("#creativetable").append("<tr><td><textarea name=content"+msg+" cols=60 rows=3>"+unescape(content)+"</textarea></td><td valign=top><input type=text name=creativename"+msg+" value=\""+unescape(creativename)+"\" size=20></td><td valign=top><input type=radio name=creativeid value="+msg+"></td></tr>");
				f.elements['content'].value="";
				f.elements['creativename'].value="";
			} else {
				alert( msg );
			}
		}
	});
  });
});

</script>

<p> &nbsp; </p>

<form name=creative method=post action=creative >
<input type=hidden name='action' value='mulupdate'>
<input type=hidden name='campaignid' value='[% campaignid %]'>
<input type=hidden name='campaignmd5' value='[% campaignmd5 %]'>
<input type=hidden name='campaignname' value="[% campaignname %]">
<input type=hidden name='itemid' value='[% item.itemid %]'>
<input type=hidden name='itemmd5' value='[% item.itemmd5 %]'>
<input type=hidden name='itemname' value="[% item.itemname %]">

<table width=600 id=creativetable border=0>
<tr><th>Content</th><th>Creative Name</th><td><img id=creativeremove src="/uilib/comImg/delete.gif" border=0 /></td></tr>
[% FOREACH creative=item.Goto_Creative_Model_topics %]<tr>
<td><textarea name=content[% creative.creativeid %] cols=60 rows=3>[% creative.content %]</textarea></td>
<td valign=top><input type=text name=creativename[% creative.creativeid %] value="[% creative.creativename %]" size=20></td>
<td valign=top><input type=radio name=creativeid value=[% creative.creativeid %]></td>
</tr>
[% END %]
</table>

<table width=600>
<tr><td colspan=3><input type=submit value='Update'></td></tr>
<tr><td>
<textarea name=content cols=60 rows=3></textarea>
</td>
<td valign=top><input type=text name=creativename size=20></td>
<td valign=top><input type=button id=creativesubmit value='Add'></td>
</tr>
</table>

</form>


</div><!-- end <div class="ui-layout-center">-->
[% INCLUDE end.e %]
