<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd"> 
<html xmlns="http://www.w3.org/1999/xhtml"> 
  <head> 
    <title>Edit Advertiser Password</title> 
    <script src="../../../js/jquery-1.4.2.min.js"></script>
    [% INCLUDE form_ui_start.e %]
  <body> 
    <script>
      var url = new String( window.location )
      var SEARCH_STRING = "advid="
      var idx = url.indexOf( SEARCH_STRING )
      var pair = url.substring( idx, url.length )
      var advid = pair.split( '=' )[ 1 ] 

      $( document ).ready( 
        function() {
          $( '#advid' )[ 0 ].value = advid
          $( '#btnSubmit' )[ 0 ].value = "Change Password for Advertiser #" + advid
        }
      )      
    </script>
    <h2 align="center" class="curTitle">Edit Advertiser Password</h2>
    <div align="center">  
    <div id="container">
    <form action="adv" method="POST"> 
        <input id="action" name="action" type="hidden" value="updatepass" />
        <input id="advid" name="advid" type="hidden" /> 
    <div id="mainmenu">
					<ul id="tabs">
						<li>
							<a href="#EditAdvertiserPassword">Edit Advertiser Password</a>
						</li>
					</ul>
				<div>
				<div class="bar">&nbsp;</div>
				<div class="panel" id="EditAdvertiserPassword">
				<fieldset>
						<legend>EditAdvertiserPassword</legend>
					<table> 
          <tbody> 
            <tr class="form-row"> 
              <td class="field-label"> 
                <label>New password</label> 
              </td>          
              <td class="field-widget"> 
                <input id="passwd" name="passwd" type="password" /> 
              </td> 
            </tr> 
            <tr class="form-row"> 
              <td class="field-label"> 
                <label>Confirm new password</label> 
              </td>          
              <td class="field-widget"> 
                <input id="confirm" name="confirm" type="password" /> 
              </td>            
            </tr> 
            <tr class="form-row"> 
              <td class="field-label"> 
                <label>&nbsp;</label> 
              </td>          
              <td class="field-widget"> 
                <input id="btnSubmit" name="btnSubmit" type="submit" value="Update Password" />
              </td>            
            </tr>          
          </tbody> 
        </table> 
												
				</fieldset>
				</div><!--end EditAdvertiserPassword-->
				</form>
		</div><!--end container-->
   
    </div> 
    [% INCLUDE form_ui_end.e %]
  </body> 
</html>