[% INCLUDE start.e %]

[% FOREACH row IN topics %]
  <tr id="[% GET row.advid %]_[% GET row.campaignid %]" class="Row">
    <td>
      <input id="[% GET row.campaignid %]" name="[% GET row.campaignid %]" class="cb" type="checkbox" />
    </td>
    <td>
      <div style="text-align: right">[% GET row.campaignid %]</div>
      <input id="campaignid" name="campaignid" type="hidden" value="[% GET row.campaignid %]" />
    </td>
    <td>[% GET row.campaignname %]</td>
    <td style="text-align: right">[% GET row.advid %]</td>
    <td style="text-align: right">[% GET row.foreignid %]</td>
    <td style="text-align: right">[% GET row.languageid %]</td>
    <td class="Invisible">[% GET row.fl_platform %]</td> 
    <td class="Invisible">[% GET row.fl_reader %]</td>
    <td class="Invisible">[% GET row.fl_style %]</td> 
    <td class="Invisible">[% GET row.fl_vertical %]</td>
    <td>[% GET row.qa_campaign %]</td>
    <td class="Invisible" style="text-align: right">[% GET row.fl_site %]</td>
    <td>[% GET row.startx %]</td>
    <td>[% GET row.endx %]</td>
    <td class="Invisible" style="text-align: right">[% GET row.cpm_fc %]</td>
    <td class="Invisible" style="text-align: right">[% GET row.cpm_length %]</td>
    <td class="Invisible" style="text-align: right">[% GET row.cpm_throttle %]</td>
    <td class="Invisible" style="text-align: right">[% GET row.cpc_fc %]</td>
    <td class="Invisible" style="text-align: right">[% GET row.cpc_length %]</td>
    <td class="Invisible" style="text-align: right">[% GET row.cpa_fc %]</td>
    <td class="Invisible" style="text-align: right">[% GET row.cpa_length %]</td>
    <td style="text-align: center">[% GET row.channelorder %]</td>
    <td style="text-align: center">[% GET row.accessorder %]</td>
    <td style="text-align: center">[% GET row.status %]</td>
    <td class="rowData">
      <a href="campaign?action=edit&campaignid=[% GET row.campaignid %]">Edit Campaign</a>
    </td>
    <td>
      <a href="item?action=topics&campaignid=[% GET row.campaignid %]">Edit Items</a>
    </td>
    <td class="rowData">
      <a href="campaign?action=delete&campaignid=[% GET row.campaignid %]&advid=[% GET row.advid %]">Delete Campaign</a>
    </td>    
  </tr>  
[% END %]

[% INCLUDE end.e %]