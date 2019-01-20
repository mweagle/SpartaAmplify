import React from 'react';
import ReactDOM from 'react-dom';
import './index.css';
import App from './App';
import registerServiceWorker from './registerServiceWorker';
import 'whatwg-fetch'

import Amplify from 'aws-amplify'
registerServiceWorker();


// Fetch the JSON document, then return the App
function checkStatus(response) {
  if (response.status >= 200 && response.status < 300) {
    return response
  } else {
    var error = new Error(response.statusText)
    error.response = response
    throw error
  }
}

function parseJSON(response) {
  return response.json()
}
// Fetch the configuration asynchronously, then
// initialize Amplify, then get going...
window.fetch('MANIFEST.json')
  .then(checkStatus)
  .then(parseJSON)
  .then(function(data) {
    console.log('request succeeded with JSON response', data)
    Amplify.configure(data.userdata || {})
    ReactDOM.render(<App />, document.getElementById('root'));
  }).catch(function(error) {
    console.log('request failed', error)
    // TODO - render an error page...
  })
