
addEventListener("fetch", (event) => {
  event.respondWith(
    handleRequest(event.request).catch(
      (err) => new Response(err.stack, { status: 500 })
    )
  );
});

/**
 * Many more examples available at:
 *   https://developers.cloudflare.com/workers/examples
 * @param {Request} request
 * @returns {Promise<Response>}
 */
async function handleRequest(request) {
  const { pathname } = new URL(request.url);

  //const request = event.request
  const url = request.url;
  const objURL = new URL(url);
  const fetchURL = objURL.searchParams.get("url");
  //const fetchJsonHeaders = objURL.searchParams.get("headers");
  let fetchMethod = objURL.searchParams.get("method");
  if( !fetchMethod ){
    fetchMethod = "GET";
  }

  let headers = request.headers;

  const modifiedRequest = new Request(fetchURL, {
    body: request.body,
    headers: headers,
    method: request.method,
    redirect: request.redirect
  })

  //modifiedRequest.headers.append("User-Agent",clientUG);

  if (pathname.startsWith("/fetch") && fetchURL ) {
    //set new host
    const objFetchURL = new URL(fetchURL);
    modifiedRequest.headers.set("Host",objFetchURL.host);

    const resp = await fetch(modifiedRequest);
    return resp;
  }

  return new Response(JSON.stringify({ MSG:"Not support." }), {
    headers: { "Content-Type": "application/json" },
  });
}
