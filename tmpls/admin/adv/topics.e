[% INCLUDE start.e %]

[% FOREACH u IN topics %]
  <tr id="[% GET u.advid %]">
    <td>
      <a href="campaign?action=topics&advid=[% GET u.advid %]&company_esc=[% GET u.company_esc %]">[% GET u.advid %]</a>
      <input id="advid" name="advid" type="hidden" value="[% GET u.advid %]" />
    </td>
    <td>
      <div align="align: right">[% GET u.email %]</div>
      
    </td>
    <td><a target=_blank href='[% GET u.url %]'>[% GET u.company %]</a></td>    
    <td class="Invisible">[% GET u.contact %]</td>          
    <td class="Invisible">[% GET u.phone %]</td>    
    <td>[% GET u.created %]</td>              
    <td style="text-align: center"><img src='/uilib/comImg/[% u.status %].png' /></td>
    <td class="userData">
      <a href="adv?action=edit&advid=[% GET u.advid %]"><img src="/uilib/comImg/editor.gif" border=0 alt="Edit" /></a>&nbsp;&nbsp;
<a href="adv?action=editpass&advid=[% GET u.advid %]">
<img src="/uilib/comImg/passwd.jpg" border=0 alt="Edit Password" /></a>
&nbsp;&nbsp;
<a href="javascript:execConfirmHrefDelete('adv?action=delete&advid=[% GET u.advid %]','e')"><img src="/uilib/comImg/delete.gif" border=0 alt="Delete" /></a>
    </td>    
  </tr>  
[% END %]

[% INCLUDE end.e %]
