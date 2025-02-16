[% INCLUDE start.e %]
<div class="ui-layout-west">
    <ul>
            <li><a href="site?action=edit&siteid=[% siteid %]">[% sitename %]</a>
            <p></p>
			<ul><li><a href="slot?action=edit&slotid=[% slotid %]&siteid=[% siteid %]&sitemd5=[% sitemd5 %]&sitename_esc=[% sitename_esc %]">[% slotname %]</a>
			<p></p>
				<ul>
                    <li>Trafficking</li>
            </ul></li>
        </ul></li>
    </ul>
</div>
<div class="ui-layout-center"> 

<h3>Items for [% slotname %]</h3>

<form action="weight" method=post>
<input type="hidden" name="action" value="mulinsert" />
<input type="hidden" name="slotid" value="[% slotid %]" />
<input type="hidden" name="slotmd5" value="[% slotmd5 %]" />
<input type="hidden" name="slotname" value="[% slotname %]" />
<input type="hidden" name="siteid" value="[% siteid %]" />
<input type="hidden" name="sitemd5" value="[% sitemd5 %]" />
<input type="hidden" name="sitename" value="[% sitename %]" />

<table border=0>
<tr>
<th>Campaign</th>
<th>Item</th>
<th>Start</th>
<th>End</th>
<th>Price</th>
<th>Priority</th>
<th>Scale</th>
</tr>
[% FOREACH item IN startnew %]<tr>
<td>[% item.campaignname %]</td>
<td>[% item.itemname %]</td>
<td><small>[% item.startx %]</small></td>
<td><small>[% item.endx %]</small></td>
<td>[% item.costtype %]: [% item.cost %]</td>
<td><select size=1 name=priority[% item.itemid %]>
<option value=0>House</option>
<option value=1>Low Network</option>
<option value=2>High Network</option>
<option value=3>RTB</option>
<option value=4>Non-Guaranteed</option>
<option value=5>Premium</option>
<option value=6>Exclusive</option>
</select></td>
<td><input type=text name=scale[% item.itemid %] value='50' size=2 maxlength=2></td>
</tr>[% END %]
</table>
<input type=submit value='Update Trafficking'>
</form>

</div>
[% INCLUDE end.e %]
