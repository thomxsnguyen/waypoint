// A minimal external Google Maps double. DayMap's actual marker creation and
// selection code still runs, including its shared stable activity identifiers.
export const mapsScript = `
class TestMap {
 constructor(element) { this.element=element; element.setAttribute('data-testid','google-map'); }
 setCenter() {} setZoom() {} fitBounds() {} panTo() {}
}
class Bounds { extend() {} getCenter() { return {lat:35,lng:139}; } }
class Marker {
 constructor(options) { this.position=options.position; this.content=options.content; this.map=options.map; }
 set map(value) { if(value) value.element.appendChild(this.content); else this.content?.remove(); }
}
window.google={maps:{Map:TestMap,LatLngBounds:Bounds,marker:{AdvancedMarkerElement:Marker},importLibrary:async()=>({})}};
window.waypointMapsReady();
`;
