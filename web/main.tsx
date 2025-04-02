import "preact/debug";
import { render } from 'preact';

const Root = () => <div>Hello world!</div>

render(<Root />, document.getElementById("root"));
console.log("running!");

