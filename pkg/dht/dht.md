## Notes:
- A starter node is the first node in the ring. It is the only node that joins the system with `n.Join(nil)`
- Each node, except for the starter node, always has a successor other than itself.