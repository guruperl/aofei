[% INCLUDE start.e %]

[% FOREACH row IN topics %]
  <tr id="[% GET row.advid %]_[% GET row.campaignid %]">
    <td>
      <a href="item?action=topics&campaignid=[% GET row.campaignid %]&campaignname_esc=[% GET row.campaignname %]&advid=[% advid %]&company_esc=[% company_esc %]">[% GET row.campaignid %]</a>
      <input id="campaignid" name="campaignid" type="hidden" value="[% GET row.campaignid %]" />
    </td>
    <td>[% GET row.campaignname %]</td>
    <td>[% GET row.qa_campaign %]</td>
    <td>[% GET row.fl_site %]</td>
    <td nowrap>[% GET row.startx %]</td>
    <td nowrap>[% GET row.endx %]</td>
    <td class="Invisible" style="text-align: right">[% GET row.cpm_fc %]</td>
    <td style="text-align: center"><img src='/uilib/comImg/[% row.status %].png' /></td>
    <td class="rowData">
     <a href="campaign?action=edit&campaignid=[% GET row.campaignid %]"><img src="/uilib/comImg/editor.gif" border=0 alt="Edit Campaign" /></a>&nbsp;&nbsp;<a href="javascript:execConfirmHrefDelete('campaign?action=delete&campaignid=[% GET row.campaignid %]&advid=[% GET row.advid %]','e')"><img src="/uilib/comImg/delete.gif" border=0 alt="Delete Campaign" /></a>
    </td>    
  </tr>  
[% END %]
[% INCLUDE end.e %]
