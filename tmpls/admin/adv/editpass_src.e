<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd"> 
<html xmlns="http://www.w3.org/1999/xhtml"> 
  <head> 
    <title>Edit Advertiser Password</title> 
    <script src="../../../js/jquery-1.4.2.min.js"></script>
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
    <h2 align="center">Edit Advertiser Password</h2>
    <div align="center">      
      <form action="adv" method="POST"> 
        <input id="action" name="action" type="hidden" value="updatepass" /> 
        <input id="advid" name="advid" type="hidden" /> 
        <table> 
          <tbody> 
            <tr> 
              <td> 
                <label>New password</label> 
              </td>          
              <td> 
                <input id="passwd" name="passwd" type="password" /> 
              </td> 
            </tr> 
            <tr> 
              <td> 
                <label>Confirm new password</label> 
              </td>          
              <td> 
                <input id="confirm" name="confirm" type="password" /> 
              </td>            
            </tr>          
          </tbody> 
        </table> 
        <br/> 
        <input id="btnSubmit" name="btnSubmit" type="submit" /> 
      </form>  
    </div> 
  </body> 
</html>